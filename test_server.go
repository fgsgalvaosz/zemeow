package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting test server on :8080...")
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ZeMeow API is running!")
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "ok", "service": "zemeow"}`)
	})
	
	fmt.Println("Server started successfully!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
