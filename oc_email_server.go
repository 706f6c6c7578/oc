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
	password    = "onioncouriermailer"
	defaultFrom = "Onion Courier <noreply@your.domain>"
	host        = "smtp.your.domain"
	port        = "2525"
	torProxy    = "127.0.0.1:9050"
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

	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Error reading file:", err)
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	to, customFrom, err := extractHeaders(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if to == "" {
		http.Error(w, "Missing To email address", http.StatusBadRequest)
		return
	}

	fromHeader := defaultFrom
	if customFrom != "" {
		fromHeader = customFrom
	}

	err = sendMail(content, to, fromHeader)
	if err != nil {
		log.Println("Error sending mail:", err)
		http.Error(w, "Error sending mail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File received and sent.\nNo data is stored or logged by Onion Courier.\n")
}

func extractHeaders(content []byte) (to string, from string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	toRe := regexp.MustCompile(`(?i)^To:\s*(.*)$`)
	fromRe := regexp.MustCompile(`(?i)^From:\s*(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		if match := toRe.FindStringSubmatch(line); match != nil {
			to = strings.TrimSpace(match[1])
		}
		if match := fromRe.FindStringSubmatch(line); match != nil {
			from = strings.TrimSpace(match[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error reading content: %v", err)
	}

	return to, from, nil
}

func sendMail(message []byte, to string, from string) error {
    hasFromHeader := bytes.Contains(bytes.ToLower(message), []byte("from:"))
    
    var headers []byte
    if !hasFromHeader {
        headers = []byte(fmt.Sprintf("From: %s\r\nNewsgroups: %s\r\n", from, to))
    }
    
    finalMessage := append(headers, message...)
    if !bytes.HasSuffix(finalMessage, []byte("\r\n")) {
        finalMessage = append(finalMessage, []byte("\r\n")...)
    }

    emailOnly := from
    if strings.Contains(from, "<") {
        parts := strings.Split(from, "<")
        emailOnly = strings.TrimSuffix(parts[1], ">")
    }

	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	customDialer := func(network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

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

	if err = c.Mail(emailOnly); err != nil {
		return fmt.Errorf("error Mail: %v", err)
	}

	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("error Rcpt: %v", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("error Data: %v", err)
	}

	_, err = w.Write(finalMessage)
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