package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/kolsmer/url-availability-checking/internal/storage"
)


type mockStorage struct {
    mu            sync.Mutex
    results       map[string]storage.URLCheck
    status        string
    resultsAtDone int
}

func (m *mockStorage) AddURLs(ctx context.Context, urls []string) (string, error) {
	return "test-id", nil
}
func (m *mockStorage) AddResult(ctx context.Context, id string, url string, result storage.URLCheck) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.results[url] = result
    return nil
}
func (m *mockStorage) GetStatusAndResult(ctx context.Context, id string) (storage.Check, error) {
	return storage.Check{
		Status: "completed",
		URLs: map[string]storage.URLCheck{
			"http://example.com": {HTTPStatus: 200},
		},
	}, nil
	}
func (m *mockStorage) UpdateStatus(ctx context.Context, id string, status string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.status = status
    if status == "done" {
        m.resultsAtDone = len(m.results)
    }
    return nil
}		



func TestCheckURL(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)
		t.Cleanup(server.Close)
		status, _, err := checkURL(context.Background(), server.Client(), server.URL)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("Expected status %d, got %d", http.StatusOK, status)
		}
	})
	t.Run("returns 404", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
		)
		t.Cleanup(server.Close)
		status, _, err := checkURL(context.Background(), server.Client(), server.URL)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if status != http.StatusNotFound {
			t.Fatalf("Expected status %d, got %d", http.StatusNotFound, status)
		}
	})
	
}

func TestRunCheck(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(server.Close)
	mock := &mockStorage{
		results: make(map[string]storage.URLCheck),
	}
	ctx := context.Background()
	service := NewURLService(ctx, mock, 2)
	urls := []string{server.URL + "/one", server.URL + "/two"}
	err := service.runCheck(ctx, "test-id", urls)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if mock.status != "done" {
		t.Fatalf("expected done, got %q", mock.status)
	}

	if mock.resultsAtDone != 2 {
		t.Fatalf("expected 2 results before done, got %d", mock.resultsAtDone)
	}

	for _, url := range urls {
		result, ok := mock.results[url]
		if !ok {
			t.Fatalf("result for %s was not saved", url)
		}

		if result.HTTPStatus != http.StatusOK || result.Err != nil {
			t.Fatalf("unexpected result for %s: %+v", url, result)
		}
	}
	
}