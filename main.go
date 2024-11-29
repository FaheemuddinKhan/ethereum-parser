package main

import (
	"context"
	"ethereum-parser/handler"
	"ethereum-parser/notifier"
	"ethereum-parser/parser"
	"ethereum-parser/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize dependencies
	nfy := notifier.NewDunnyNotifier()
	storage := storage.NewInMemoryStorage()
	apiURL := "ethereum-rpc.publicnode.com"
	parserService := parser.NewParserService(storage, apiURL, nfy)
	parserHandler := handler.NewParserHandler(parserService)

	// Listen for new block creations
	parserService.Start()

	// Initialize router
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/current-block", parserHandler.GetCurrentBlockHandler).Methods("GET")
	router.HandleFunc("/subscribe", parserHandler.SubscribeHandler).Methods("POST")
	router.HandleFunc("/transactions", parserHandler.GetTransactionsHandler).Methods("GET")

	// Create a server instance
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Println("Server is listening on :8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("could not listen on :8080: %v", err)
		}
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	log.Println("Received interrupt signal, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully shut down the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully shut down.")
}
