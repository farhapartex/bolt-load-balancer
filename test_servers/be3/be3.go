package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, " Response from Backend Server 3 (Port 8083)\n")
		fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Headers: %v\n", r.Header)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintf(w, "Backend 3 is healthy")
	})

	http.HandleFunc("/api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "Login processed by Backend 3", 
            "server": "backend-3", 
            "port": 8083,
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	http.HandleFunc("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "User profile from Backend 3", 
            "server": "backend-3", 
            "port": 8083,
            "user_id": 12345,
            "username": "testuser",
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	http.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "message": "Users list from Backend 3", 
            "server": "backend-3", 
            "port": 8083,
            "users": ["user7", "user8", "user9"],
            "timestamp": "%s"
        }`, time.Now().Format("2006-01-02 15:04:05"))
	})

	fmt.Println(" Backend Server 3 starting on port 8083...")
	fmt.Println("   Health check: http://localhost:8083/health")
	fmt.Println("   API endpoints: http://localhost:8083/api/v1/*")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
