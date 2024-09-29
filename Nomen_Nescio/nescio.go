package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/smtp"
	"os"

	"golang.org/x/net/proxy"
)

func main() {
	// contacting server as
	from := "onion@onion.onion"

	// sending message to
	to := "remailer@dizum.com"

	// server we send message through
	host := "smtp.dizum.com"
	port := "2525"

	// Tor SOCKS5 proxy address (standard Tor port)
	torProxy := "127.0.0.1:9050"

	// check if there is piped input
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		fmt.Println("Usage: nescio < message.txt")
		os.Exit(1)
	}

	message, _ := ioutil.ReadAll(os.Stdin)

	// Create a SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		fmt.Println("Error creating SOCKS5 dialer:", err)
		os.Exit(1)
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
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		fmt.Println("Error creating SMTP client:", err)
		os.Exit(1)
	}

	err = c.StartTLS(tlsConfig)
	if err != nil {
		fmt.Println("Error starting TLS:", err)
		os.Exit(1)
	}

	if err = c.Mail(from); err != nil {
		fmt.Println("Error Mail:", err)
		os.Exit(1)
	}

	if err = c.Rcpt(to); err != nil {
		fmt.Println("Error Rcpt:", err)
		os.Exit(1)
	}

	w, err := c.Data()
	if err != nil {
		fmt.Println("Error Data:", err)
		os.Exit(1)
	}

	_, err = w.Write(message)
	if err != nil {
		fmt.Println("Error Write:", err)
		os.Exit(1)
	}

	err = w.Close()
	if err != nil {
		fmt.Println("Error Close:", err)
		os.Exit(1)
	}

	c.Quit()

	fmt.Println("Message Sent!")
}
