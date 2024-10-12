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
	"regexp"
	"strings"

	"golang.org/x/net/proxy"
)

const (
	nickname = "server's nckname"
	password = "secretPassword" // Set your desired password here
	from     = "Onion Courier <noreply@your.domain.org>"
	host     = "smtp.your.domain.org"
	port     = "2525"
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

	to, err := extractToHeader(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if to == "" {
		http.Error(w, "Missing To email address", http.StatusBadRequest)
		return
	}

	// Send the content via email and get the session log
	sessionLog, err := sendMail(content, to)
	if err != nil {
		log.Println("Error sending mail:", err)
		http.Error(w, "Error sending mail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Output to the client
	fmt.Fprintf(w, "File received and sent!\n\nSMTP Session Log:\n%s", sessionLog)
}

func extractToHeader(content []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	re := regexp.MustCompile(`(?i)^To:\s*(.*)$`) // Case insensitive match for "To:"

	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); match != nil {
			return strings.TrimSpace(match[1]), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading content: %v", err)
	}
	return "", nil
}

func sendMail(message []byte, to string) (string, error) {
	var sessionLog bytes.Buffer

	// Create a SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("error creating SOCKS5 dialer: %v", err)
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
		return "", fmt.Errorf("error connecting to server: %v", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return "", fmt.Errorf("error creating SMTP client: %v", err)
	}

	sessionLog.WriteString("Connected to SMTP server smtp.your.domain.org\n")

	err = c.StartTLS(tlsConfig)
	if err != nil {
		return "", fmt.Errorf("error starting TLS: %v", err)
	}
	sessionLog.WriteString("TLS started\n")

	if err = c.Mail(from); err != nil {
		return "", fmt.Errorf("error Mail: %v", err)
	}
	sessionLog.WriteString(fmt.Sprintf("From: %s\n", from))

	if err = c.Rcpt(to); err != nil {
		return "", fmt.Errorf("error Rcpt: %v", err)
	}
	sessionLog.WriteString(fmt.Sprintf("To: %s\n", to))

	w, err := c.Data()
	if err != nil {
		return "", fmt.Errorf("error Data: %v", err)
	}
	sessionLog.WriteString("Data command sent\n")

	_, err = w.Write(message)
	if err != nil {
		return "", fmt.Errorf("error Write: %v", err)
	}
	sessionLog.WriteString("Message body sent\n")

	err = w.Close()
	if err != nil {
		return "", fmt.Errorf("error Close: %v", err)
	}
	sessionLog.WriteString("Message body closed\n")

	c.Quit()
	sessionLog.WriteString("QUIT command sent\n")
	fmt.Fprintf(&sessionLog, "No data is stored or logged by Onion Courier %s.\n", nickname)

	return sessionLog.String(), nil
}
