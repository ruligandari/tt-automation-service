package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tiktok-automation-service/internal/client"
	"tiktok-automation-service/internal/handler"
	"tiktok-automation-service/internal/service"
)

func main() {
	// Initialize HTTP Client
	httpClient := client.NewHTTPClient()

	// Initialize services
	tiktokSvc := service.NewTiktokService(httpClient)
	whatsappSvc := service.NewWhatsAppService(httpClient)

	// Initialize handler
	webhookHandler := handler.NewWebhookHandler(tiktokSvc, whatsappSvc)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", webhookHandler.HandleWebhook)
	mux.HandleFunc("/health", handler.HandleHealth)

	// Create server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Channel to listen for errors during startup
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Println("Starting server on :8080")
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for termination signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking for signal or error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server failed to start: %v", err)

	case sig := <-shutdown:
		log.Printf("Start shutdown... Signal: %v", sig)

		// Give the server 5 seconds to finish processing requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			if err := srv.Close(); err != nil {
				log.Fatalf("Force shutdown failed: %v", err)
			}
		}
		log.Println("Server stopped gracefully")
	}
}
