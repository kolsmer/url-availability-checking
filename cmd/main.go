package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/kolsmer/url-availability-checking/internal/handler"
	"github.com/kolsmer/url-availability-checking/internal/service"
	"github.com/kolsmer/url-availability-checking/internal/storage"
)

func main() {
	maxConcurrentChecks := flag.Int("concurrent", 5, "Maximum number of concurrent URL checks")
	flag.Parse()
	if *maxConcurrentChecks <= 0 {
		log.Fatal("Maximum number of concurrent URL checks must be a positive integer")
	}
	appCtx := context.Background()
	storage := storage.NewInternalStorage()
	service := service.NewURLService(appCtx, storage, *maxConcurrentChecks)
	handler := handler.NewURLHandler(service)
	

	http.HandleFunc("/checks", handler.CheckAvailability)
	http.HandleFunc("/checks/{id}", handler.GetStatusAndResult)

	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}