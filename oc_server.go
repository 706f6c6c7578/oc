package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const password = "secretPassword" // Set your desired password here

var filePath string

func init() {
	flag.StringVar(&filePath, "p", ".", "Path to save uploaded files")
	flag.Parse()
}

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check the password
	if r.Header.Get("X-Password") != password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create the destination file with the specified path
	dstPath := filepath.Join(filePath, header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Output to stderr with timestamp and username (if provided)
	currentTime := time.Now().Format("15:04:05")
	username := r.Header.Get("X-Username")
	if username == "" {
		username = "Anonymous"
	}
	fmt.Fprintf(os.Stderr, "File received: %s at %s by %s\n", header.Filename, currentTime, username)

	// Output to the client
	fmt.Fprintf(w, "File %s successfully uploaded!", header.Filename)
}
