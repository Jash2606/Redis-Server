package routes

import (
	"fmt"
	"net/http"
	"time"

	"Key_Value_Cache_Ass/controllers"
)

// RegisterRoutes registers all endpoints with optimized server settings.
func RegisterRoutes() *http.Server {
	// Set up routes
	fmt.Print("Registering routes...")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Print("Received request to /")
		w.Write([]byte("yes"))
	})
	http.HandleFunc("/put", controllers.PutCache)
	http.HandleFunc("/get", controllers.GetCache)
	
	fmt.Println("Routes registered")
	// Create optimized server
	server := &http.Server{
		Addr:         ":7171",
		Handler:      http.DefaultServeMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		MaxHeaderBytes: 1 << 20,  
	}
	
	return server
}
