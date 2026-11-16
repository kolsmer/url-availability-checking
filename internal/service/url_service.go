package service

import (
	"context"
	"net/http"
	"time"
	"sync"

	"github.com/kolsmer/url-availability-checking/internal/storage"
)
var (
	maxConcurrentChecks = 5
	maxRequestTimeout   = 3 * time.Second
)
type URLService struct {
	storage storage.Storage
}

func NewURLService(storage storage.Storage) *URLService {
	return &URLService{
		storage: storage,
	}
}

func (s *URLService) CheckAvailability(ctx context.Context, urls []string)  (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if len(urls) == 0 {
		return "", nil
	}
	
	id, err := s.storage.AddURLs(ctx, urls)

	if err != nil {
		return "", err
	}
	client := &http.Client{}

	semaphore := make(chan struct{}, maxConcurrentChecks)
	var wg sync.WaitGroup



	for _, url := range urls {
		semaphore <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			statusCode, responseTime, err := checkURL(ctx, client, url)
			urlCheck := storage.URLCheck{
				HTTPStatus:   statusCode,
				ResponseTime: responseTime,
				Err:          err,
			}
			s.storage.AddResult(ctx, id, url, urlCheck)
		}(url)

	}

	wg.Wait()

	err = s.storage.UpdateStatus(ctx, id, "completed")


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

func (s *URLService) GetStatusAndResult(ctx context.Context, id string) (storage.Check, error) {
	if ctx.Err() != nil {
		return storage.Check{}, ctx.Err()
	}

	return s.storage.GetStatusAndResult(ctx, id)
}
