package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a pipe to pass the data to the sendArticle function
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	// Write the file content to the pipe
	go func() {
		_, err := io.Copy(writer, file)
		if err != nil {
			log.Println("Error writing to pipe:", err)
		}
		writer.Close()
	}()

	// Call the sendArticle function and get the session log
	sessionLog, err := sendArticle(reader)
	if err != nil {
		log.Println("Error sending article:", err)
		http.Error(w, "Error sending article: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Output to stderr with timestamp and username (if provided)
	currentTime := time.Now().Format("15:04:05")
	username := r.Header.Get("X-Username")
	if username == "" {
		username = "Anonymous"
	}
	fmt.Fprintf(os.Stderr, "File %s received at %s by %s\n", header.Filename, currentTime, username)

	// Output to the client
	fmt.Fprintf(w, "File %s received and sent!\n\nNNTP Session Log:\n%s", header.Filename, sessionLog)
}

func sendArticle(reader io.Reader) (string, error) {
	var sessionLog bytes.Buffer

	// Create a SOCKS5 dialer using the Tor proxy
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	// Establish a connection to the NNTP server through Tor
	conn, err := dialer.Dial("tcp", server)
	if err != nil {
		return "", fmt.Errorf("error connecting to the server through Tor: %v", err)
	}
	defer conn.Close()

	// Read greeting from server
	bufReader := bufio.NewReader(conn)
	response, err := bufReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error when reading the server greeting: %v", err)
	}
	sessionLog.WriteString("Server: " + response)

	// Send POST command
	fmt.Fprintf(conn, "POST\r\n")
	sessionLog.WriteString("OCServ: POST\r\n")
	response, err = bufReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading the POST response: %v", err)
	}
	sessionLog.WriteString("Server: " + response)

	if !strings.HasPrefix(response, "340") {
		return "", fmt.Errorf("server does not accept POST")
	}

	// Read and send articles from the reader
	sessionLog.WriteString("OCServ: Data not stored nor logged by Usenet Onion Gateway.\n")
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(conn, "%s\r\n", line)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading the input: %v", err)
	}

	// Send end of article
	fmt.Fprintf(conn, ".\r\n")
	sessionLog.WriteString("OCServ: .\r\n")

	// Read response from the server
	response, err = bufReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading the server response: %v", err)
	}
	sessionLog.WriteString("Server: " + response)

	// Send QUIT command
	fmt.Fprintf(conn, "QUIT\r\n")
	sessionLog.WriteString("OCServ: QUIT\r\n")

	// Read the final server response
	finalResponse, err := ioutil.ReadAll(bufReader)
	if err != nil {
		return "", fmt.Errorf("error reading the final server response: %v", err)
	}
	sessionLog.Write(finalResponse)

	return sessionLog.String(), nil
}
