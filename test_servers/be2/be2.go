package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, " Response from Backend Server 2 (Port 8082)\n")
		fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Headers: %v\n", r.Header)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintf(w, "Backend 2 is healthy")
	})

	http.HandleFunc("/api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "Login processed by Backend 2", 
            "server": "backend-2", 
            "port": 8082,
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	http.HandleFunc("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "User profile from Backend 2", 
            "server": "backend-2", 
            "port": 8082,
            "user_id": 12345,
            "username": "testuser",
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	http.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "Users list from Backend 2", 
            "server": "backend-2", 
            "port": 8082,
            "users": ["user4", "user5", "user6"],
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	fmt.Println(" Backend Server 2 starting on port 8082...")
	fmt.Println("   Health check: http://localhost:8082/health")
	fmt.Println("   API endpoints: http://localhost:8082/api/v1/*")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
