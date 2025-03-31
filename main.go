package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"Key_Value_Cache_Ass/controllers"
	"Key_Value_Cache_Ass/routes"
)

func main() {
	// Register routes and get optimized server
	server := routes.RegisterRoutes()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Println("Starting Key-Value Cache service on port 7171...")
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("Server failed:", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Clean up resources
	controllers.CacheInstance.Close()
	log.Println("Server gracefully stopped")
}
