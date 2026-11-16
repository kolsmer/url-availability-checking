package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kolsmer/url-availability-checking/internal/storage"
)
var (
	maxRequestTimeout   = 3 * time.Second
	ErrInvalidURLs      = errors.New("invalid URL list")
)
type URLService struct {
	appCtx context.Context
	semaphore chan struct{}
	storage storage.Storage
	client *http.Client
}

func NewURLService(appCtx context.Context, storage storage.Storage, maxConcurrentChecks int) *URLService {
	return &URLService{
		appCtx: appCtx,
		storage: storage,
		semaphore: make(chan struct{}, maxConcurrentChecks),
		client: &http.Client{},
	}
}

func (s *URLService) CheckAvailability(ctx context.Context, urls []string)  (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if len(urls) == 0 {
		return "", fmt.Errorf("%w: at least one URL is required", ErrInvalidURLs)
	}
	for _, address := range urls {
		parsed, err := url.Parse(address)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			return "", fmt.Errorf("%w: %q must be http or https URL", ErrInvalidURLs, address)
		}
	}
	
	id, err := s.storage.AddURLs(ctx, urls)

	if err != nil {
		return "", err
	}
	
	checkURLs := append([]string(nil), urls...)
	go func() {
		if err := s.runCheck(s.appCtx, id, checkURLs); err != nil {
			log.Printf("Error running URL check for ID %s: %v", id, err)
		}
	}()
	return id, nil
}

func checkURL(ctx context.Context, client *http.Client, url string) (int, time.Duration, error) {
	checkCtx, cancel := context.WithTimeout(ctx, maxRequestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(checkCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}
	start := time.Now()
	response, err := client.Do(request)
	resTime := time.Since(start)

	if err != nil {
		return 0, resTime, err
	}
	defer response.Body.Close()
	return response.StatusCode, resTime, nil
}

func (s *URLService) runCheck(ctx context.Context, id string, urls []string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	
	var wg sync.WaitGroup
	var errChan = make(chan error, len(urls))
	for _, url := range urls {
		select {
			case s.semaphore <- struct{}{}:
				wg.Add(1)
				go func(url string) {
					defer wg.Done()
					defer func() { <-s.semaphore }()

					statusCode, responseTime, err := checkURL(ctx, s.client, url)
					urlCheck := storage.URLCheck{
						HTTPStatus:   statusCode,
						ResponseTime: responseTime,
						Err:          err,
					}
					err = s.storage.AddResult(ctx, id, url, urlCheck)
					if err != nil {
						errChan <- err
						return
					}
				}(url)
			case <-ctx.Done():
				wg.Wait()
				return ctx.Err()
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return s.storage.UpdateStatus(ctx, id, "done")
}

func (s *URLService) GetStatusAndResult(ctx context.Context, id string) (storage.Check, error) {
	if ctx.Err() != nil {
		return storage.Check{}, ctx.Err()
	}

	return s.storage.GetStatusAndResult(ctx, id)
}
