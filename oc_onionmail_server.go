package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	
	"golang.org/x/net/proxy"
)

const (
	password = "onioncourier2onionmail" // Set your desired password here
	from     = "bounce@onion.onion"
	host     = "gtfcy37qyzor7kb6blz2buwuu5u7qjkycasjdf3yaslibkbyhsxub4yd.onion"
	port     = "25"
	torProxy = "127.0.0.1:9050"
)

func main() {
	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8083")
	http.ListenAndServe(":8083", nil)
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

	// Read the file content
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Error reading file:", err)
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	// Extract recipient from content headers
	to, messageContent, err := extractRecipient(content)
	if err != nil {
		log.Println("Error extracting recipient:", err)
		http.Error(w, "Error extracting recipient: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Send the content via email
	err = sendMail(to, messageContent)
	if err != nil {
		log.Println("Error sending mail:", err)
		http.Error(w, "Error sending mail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Output to the client
	fmt.Fprintf(w, "File received and sent.\nNo data is stored or logged by Onion Courier.\n")
}

func extractRecipient(content []byte) (string, []byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var to string
	var headerLines []string

	// Scan headers
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		headerLines = append(headerLines, line)
		if strings.HasPrefix(strings.ToLower(line), "to:") {
			to = strings.TrimSpace(strings.TrimPrefix(line, "To:"))
			to = strings.TrimSpace(strings.TrimPrefix(to, "to:"))
		}
	}

	if to == "" {
		return "", nil, fmt.Errorf("no 'To:' header found in the message")
	}

	// Reconstruct the message with headers and body
	message := append([]byte(strings.Join(headerLines, "\r\n")+"\r\n\r\n"), content[len(strings.Join(headerLines, "\n"))+1:]...)
	
	return to, message, nil
}

func sendMail(to string, message []byte) error {
	// Create a SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	// Create a custom dialer function
	customDialer := func(network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}

	// Create a custom TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Connect to the server using the custom dialer
	conn, err := customDialer("tcp", host+":"+port)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("error creating SMTP client: %v", err)
	}

	err = c.StartTLS(tlsConfig)
	if err != nil {
		return fmt.Errorf("error starting TLS: %v", err)
	}

	if err = c.Mail(from); err != nil {
		return fmt.Errorf("error Mail: %v", err)
	}

	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("error Rcpt: %v", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("error Data: %v", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("error Write: %v", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("error Close: %v", err)
	}

	c.Quit()

	return nil
}

