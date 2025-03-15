package main

import (
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "net/smtp"
    
    "golang.org/x/net/proxy"
)

type EmailDetails struct {
    From    string
    To      string
    Content []byte
}

const (
    password = "onioncourierm2ngateway"
    from     = "onion@onion.onion"
    fromHeader = "Onion Courier <noreply@oc2mx.net>"
    to       = "mail2news@oc2mx.net"
    host     = "hal.oc2mx.net"
    port     = "2525"
    torProxy = "127.0.0.1:9050"
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

    customFrom := r.Header.Get("From")
    if customFrom == "" {
        customFrom = fromHeader
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

    email := EmailDetails{
        From:    customFrom,
        To:      to,
        Content: content,
    }

    err = sendMail(email)
    if err != nil {
        log.Println("Error sending mail:", err)
        http.Error(w, "Error sending mail: "+err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "File received and sent.\nNo data is stored or logged by Onion Courier.\n")
}

func sendMail(email EmailDetails) error {
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

    if err = c.Mail(from); err != nil {
        return fmt.Errorf("error Mail: %v", err)
    }

    if err = c.Rcpt(email.To); err != nil {
        return fmt.Errorf("error Rcpt: %v", err)
    }

    w, err := c.Data()
    if err != nil {
        return fmt.Errorf("error Data: %v", err)
    }

    headers := fmt.Sprintf("From: %s\r\n", email.From)

    _, err = w.Write([]byte(headers))
    if err != nil {
        return fmt.Errorf("error writing headers: %v", err)
    }

    _, err = w.Write(email.Content)
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