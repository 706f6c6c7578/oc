package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "sync"
)

func main() {
    listener, err := net.Listen("tcp", "127.0.0.1:8083")
    if err != nil {
        panic(err)
    }
    defer listener.Close()

    fmt.Println("Server is running and waiting for connections...")

    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Error accepting connection:", err)
            continue
        }
        fmt.Println("Client connected.")
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

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
        fmt.Printf("Client: %s\n", scanner.Text())
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
