package handler

import (
	"errors"
	"encoding/json"
	"net/http"

	"github.com/kolsmer/url-availability-checking/internal/storage"
	"github.com/kolsmer/url-availability-checking/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}



func (h *URLHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var requestURLs struct {
		URLs []string `json:"urls"`
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&requestURLs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(requestURLs.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}
	id, err := h.service.CheckAvailability(r.Context(), requestURLs.URLs)
	if err != nil {
		if errors.Is(err, storage.ErrIDNotFound) {
			http.Error(w, "ID not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error checking URL availability: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (h *URLHandler) GetStatusAndResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	check, err := h.service.GetStatusAndResult(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrIDNotFound) {
			http.Error(w, "ID not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error getting status and result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}