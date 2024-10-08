package main

import (
	"fmt"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	message := os.Getenv("CLUSTER")
	if message == "" {
		message = "Default Message"
	}

	fmt.Fprintf(w, "%s\n", message)
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Serving on port 8080")
	http.ListenAndServe(":8080", nil)
}

