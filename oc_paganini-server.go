package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/proxy"
)

const (
	password = "secretPassword" // Set your desired password here
	server   = "bofhteamhroxbmd6pxbjrg6egqrnnu2vj7vlxpcnb3ypk56devuyj6yd.onion:119" // Replace this with your NNTP server and port
	torProxy = "127.0.0.1:9050" // Default Tor SOCKS proxy address
)

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8082")
	http.ListenAndServe(":8082", nil)
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
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Call the sendArticle function
	err = sendArticle(file)
	if err != nil {
		log.Println("Error sending article:", err)
		http.Error(w, "Error sending article: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Output to the client
	fmt.Fprintf(w, "File received and sent.\nNo data is stored or logged by Onion Courier.")
}

func sendArticle(reader io.Reader) error {
	// Create a SOCKS5 dialer using the Tor proxy
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	// Establish a connection to the NNTP server through Tor
	conn, err := dialer.Dial("tcp", server)
	if err != nil {
		return fmt.Errorf("error connecting to the server through Tor: %v", err)
	}
	defer conn.Close()

	// Read greeting from server
	bufReader := bufio.NewReader(conn)
	response, err := bufReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error when reading the server greeting: %v", err)
	}

	// Send POST command
	fmt.Fprintf(conn, "POST\r\n")
	response, err = bufReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading the POST response: %v", err)
	}

	if !strings.HasPrefix(response, "340") {
		return fmt.Errorf("server does not accept POST")
	}

	// Read and send articles from the reader
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(conn, "%s\r\n", line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading the input: %v", err)
	}

	// Send end of article
	fmt.Fprintf(conn, ".\r\n")

	// Read response from the server
	_, err = bufReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading the server response: %v", err)
	}

	// Send QUIT command
	fmt.Fprintf(conn, "QUIT\r\n")

	return nil
}
