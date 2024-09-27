package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/proxy"
)

const (
	server   = "paganini.bofh.team:119" // Replace this with your NNTP server and port
	torProxy = "127.0.0.1:9050"         // Default Tor SOCKS proxy address
)

func main() {
	// Create a SOCKS5 dialer using the Tor proxy
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		fmt.Println("Error creating SOCKS5 dialer:", err)
		return
	}

	// Establish a connection to the NNTP server through Tor
	conn, err := dialer.Dial("tcp", server)
	if err != nil {
		fmt.Println("Error connecting to the server through Tor:", err)
		return
	}
	defer conn.Close()

	// Read greeting from server
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error when reading the server greeting:", err)
		return
	}
	fmt.Print("Server: ", response)

	// Send POST command
	fmt.Fprintf(conn, "POST\r\n")
	response, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading the POST response:", err)
		return
	}
	fmt.Print("Server: ", response)

	if !strings.HasPrefix(response, "340") {
		fmt.Println("Server does not accept POST")
		return
	}

	// Read and send articles from the standard input
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(conn, "%s\r\n", line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading the input:", err)
		return
	}

	// Send end of article
	fmt.Fprintf(conn, ".\r\n")

	// Read response from the server
	response, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading the server response:", err)
		return
	}
	fmt.Print("Server: ", response)

	// Send QUIT command
	fmt.Fprintf(conn, "QUIT\r\n")
	io.Copy(os.Stdout, reader)
}
