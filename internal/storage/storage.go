package storage

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

var (
	ErrIDNotFound = errors.New("id not found")
)

type Storage interface {
	GetStatusAndResult(ctx context.Context, id string) (Check, error)
	AddURLs(ctx context.Context, urls []string) (string, error)
}

type URLCheck struct {
	HTTPStatus string
	responseTime int64
	Err error 
}

type Check struct{
	Status string
	URLs map[string]URLCheck
}

type InternalStorage struct {
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
	res, ok := s.memory[id]
	if !ok {
		return Check{}, ErrIDNotFound
	}

	return res, nil
}

func (s *InternalStorage) AddURLs(ctx context.Context, urls []string) (string, error){
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	id := uuid.NewString()
	check := Check{
		Status: "pending",
		URLs:   make(map[string]URLCheck),
	}
	for _, url := range urls {
		check.URLs[url] = URLCheck{}
	}
	s.memory[id] = check
	return id, nil
}

