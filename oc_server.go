package main

import (
	"crypto/rand"
	"encoding/hex"
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
	flag.StringVar(&filePath, "p", "", "Path to save uploaded files")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -p <path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -p <path>: Specify the path to save uploaded files (required)\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if filePath == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Printf("Server is running on http://localhost:8080\nFiles will be saved to: %s\n", filePath)
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

	var filename string
	if header.Filename == "message.txt" {
		// Came through an Onion Courier middleman, use random filename
		randomName, err := generateRandomFilename()
		if err != nil {
			http.Error(w, "Error generating filename", http.StatusInternalServerError)
			return
		}
		filename = randomName
	} else {
		// Use original filename
		filename = header.Filename
	}

	// Create the destination file with the specified path and filename
	dstPath := filepath.Join(filePath, filename)
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
	fmt.Fprintf(os.Stderr, "File %s received at %s by %s\n", filename, currentTime, username)

	// Output to the client
	if filename != header.Filename {
		fmt.Fprintf(w, "File received and saved as %s!", filename)
	} else {
		fmt.Fprintf(w, "File %s received!", filename)
	}
}

func generateRandomFilename() (string, error) {
	randomBytes := make([]byte, 4) // 4 bytes will give us 8 hex characters
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("m%s", hex.EncodeToString(randomBytes)[:7]), nil
}
