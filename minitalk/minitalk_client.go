package main

import (
    "bufio"
    "flag"
    "fmt"
    "net"
    "net/url"
    "os"
    "strings"
    "sync"

    "golang.org/x/net/proxy"
)

func main() {
    serverURL := flag.String("u", "", "Server URL in the format <Server-Onion-URL>:<Port>")
    flag.Parse()

    if *serverURL == "" {
        fmt.Println("Please enter the Onion URL and the port with the -u parameter.")
        os.Exit(1)
    }

    torProxyUrl, _ := url.Parse("socks5://127.0.0.1:9050")
    dialer, _ := proxy.FromURL(torProxyUrl, proxy.Direct)

    conn, err := dialer.Dial("tcp", *serverURL)
    if err != nil {
        if strings.Contains(err.Error(), "general SOCKS server failure") {
            fmt.Println("Server offline")
        } else if strings.Contains(err.Error(), "connection refused") {
            fmt.Println("Line busy")
        } else {
            fmt.Printf("Connection error: %v\n", err)
        }
        os.Exit(1)
    }
    defer conn.Close()

    // Check for server response
    reader := bufio.NewReader(conn)
    response, err := reader.ReadString('\n')
    if err != nil {
        fmt.Println("Error reading server response:", err)
        os.Exit(1)
    }
    response = strings.TrimSpace(response)

    if response != "CONNECTED" {
        fmt.Println("Unexpected server response")
        os.Exit(1)
    }

    fmt.Println("Connected to the server.")

    var wg sync.WaitGroup
    wg.Add(2)

    go receiveMessages(conn, &wg)
    go sendMessages(conn, &wg)

    wg.Wait()
}

func receiveMessages(conn net.Conn, wg *sync.WaitGroup) {
    defer wg.Done()
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        fmt.Printf("Server: %s\n", scanner.Text())
    }
}

func sendMessages(conn net.Conn, wg *sync.WaitGroup) {
    defer wg.Done()
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        message := scanner.Text()
        fmt.Printf("You: %s\n", message)
        _, err := conn.Write([]byte(message + "\n"))
        if err != nil {
            fmt.Println("Error sending message:", err)
            return
        }
    }
}
