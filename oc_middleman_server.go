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

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Error parsing multipart form", http.StatusBadRequest)
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

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	headerParts := strings.SplitN(lines[0], " ", 3)
	if len(headerParts) != 3 || headerParts[0] != "X-OC-To:" {
		http.Error(w, "Invalid header format", http.StatusBadRequest)
		return
	}

	onionAddress := headerParts[1]
	password := strings.TrimSpace(headerParts[2])
	messageBody := strings.Join(lines[2:], "\n")

	response, err := sendToOnionAddress([]byte(messageBody), onionAddress, password)
        if err != nil {
            log.Printf("Error sending to onion address: %v", err)
            http.Error(w, fmt.Sprintf("Error sending to onion address: %v", err), http.StatusInternalServerError)
            return
        }

    middlemanOnionURL := "w7t3g7oo5naebqwlezshgkgczttjn7x3re3farrzwa6bttvbnm5fcsad.onion:8084" // Replace with your actual middleman Onion URL.

    responseMsg := "\n============================\n"
    responseMsg += "File received and sent by:\n%s\n"
    responseMsg += "No data stored nor log files are written by this server.\n\n"
    responseMsg += "Target Onion Courier Response:\n%s\n"
    responseMsg += "============================\n"
    responseMsg += "%s"

    fmt.Fprintf(w, responseMsg, middlemanOnionURL, onionAddress, response)
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
