package storage

import (
	"context"
	"errors"
	"time"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrIDNotFound = errors.New("id not found")
)

type Storage interface {
	GetStatusAndResult(ctx context.Context, id string) (Check, error)
	AddURLs(ctx context.Context, urls []string) (string, error)
	AddResult(ctx context.Context, id string, url string, result URLCheck) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type URLCheck struct {
	HTTPStatus int `json:"http_status"`
	ResponseTime time.Duration `json:"response_time"`
	Err error `json:"err"`
}

type Check struct{
	Status string `json:"status"`
	URLs map[string]URLCheck `json:"urls"`
}

type InternalStorage struct {
	mu sync.RWMutex
	memory map[string]Check
}


func NewInternalStorage() *InternalStorage {
	return &InternalStorage{
		memory: make(map[string]Check),
	}
}

func (s *InternalStorage) GetStatusAndResult(ctx context.Context, id string) (Check, error){
	if ctx.Err() != nil {
		return Check{}, ctx.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	res, ok := s.memory[id]
	if !ok {
		return Check{}, ErrIDNotFound
	}

	result := Check{
		Status: res.Status,
		URLs:   make(map[string]URLCheck, len(res.URLs)),
	}
	for url, urlCheck := range res.URLs {
		result.URLs[url] = urlCheck
	}

	return result, nil
}

func (s *InternalStorage) AddURLs(ctx context.Context, urls []string) (string, error){
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	id := uuid.NewString()
	check := Check{
		Status: "running",
		URLs:   make(map[string]URLCheck),
	}
	for _, url := range urls {
		check.URLs[url] = URLCheck{}
	}
	s.mu.Lock()
	s.memory[id] = check
	s.mu.Unlock()
	return id, nil
}

func (s *InternalStorage) AddResult(ctx context.Context,id string, url string, result URLCheck) error{
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	check, ok := s.memory[id]
	if !ok {
		return ErrIDNotFound
	}
	check.URLs[url] = result
	return nil
}

func (s *InternalStorage) UpdateStatus(ctx context.Context, id string, status string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	check, ok := s.memory[id]
	if !ok {
		return ErrIDNotFound
	}
	check.Status = status
	s.memory[id] = check
	return nil
}