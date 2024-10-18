package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"golang.org/x/net/proxy"
)

const (
	serverPassword = "secretPassword" // Set your desired server password here
	maxFileSize    = 42 * 1024        // 42 KB in bytes
)

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8085")
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Password check
	if r.Header.Get("X-Password") != serverPassword {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Limit the request body size to maxFileSize
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	err := r.ParseMultipartForm(maxFileSize)
	if err != nil {
		http.Error(w, "File too large or error parsing form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file content", http.StatusInternalServerError)
		return
	}

	if len(content) > maxFileSize {
		http.Error(w, "Maximun allowed message size 42 KB!", http.StatusBadRequest)
		return
	}

	lines := strings.Split(string(content), "\n")
	var headers []string
	var messageBody string
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			messageBody = strings.Join(lines[i+1:], "\n")
			break
		}
		headers = append(headers, line)
	}

	if len(headers) == 0 {
		http.Error(w, "No valid headers found", http.StatusBadRequest)
		return
	}

	// Processing the first header
	headerParts := strings.SplitN(headers[0], " ", 3)
	if len(headerParts) != 3 || headerParts[0] != "X-OC-To:" {
		http.Error(w, "Invalid header format", http.StatusBadRequest)
		return
	}

	onionAddress := headerParts[1]
	password := strings.TrimSpace(headerParts[2])

	// Create new message without first header
	newMessage := strings.Join(append(headers[1:], "", messageBody), "\n")

	response, err := sendToOnionAddress([]byte(newMessage), onionAddress, password)
	if err != nil {
		log.Printf("Error sending to onion address: %v", err)
		http.Error(w, fmt.Sprintf("Error sending to onion address: %v", err), http.StatusInternalServerError)
		return
	}

    responseMsg := "File received and sent.\n"
    responseMsg += "No data is stored or logged by Onion Courier.\n\n"
    responseMsg += "Onion Courier Response: %s\n"

    fmt.Fprintf(w, responseMsg, response)
}

func sendToOnionAddress(message []byte, onionAddress, password string) (string, error) {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "message.txt")
	if err != nil {
		return "", fmt.Errorf("error creating form file: %v", err)
	}

	_, err = part.Write(message)
	if err != nil {
		return "", fmt.Errorf("error writing message to form file: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("error closing multipart writer: %v", err)
	}

	url := fmt.Sprintf("http://%s/upload", onionAddress)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Password", password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-OK status: %d - %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}
