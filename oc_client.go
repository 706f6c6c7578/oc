package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

var startTime time.Time

func main() {
	var username string
	var dataFile string
	var filename string
	var hideResponse bool
	flag.StringVar(&username, "u", "", "Optional username")
	flag.StringVar(&dataFile, "d", "", "File containing server addresses, ports, and passwords")
	flag.StringVar(&filename, "f", "", "File to upload")
	flag.BoolVar(&hideResponse, "h", false, "Hide server response")
	flag.Parse()

	var err error

	if dataFile != "" {
		addresses, err := readDataFile(dataFile)
		if err != nil {
			fmt.Printf("Error reading data file: %v\n", err)
			os.Exit(1)
		}
		for _, addr := range addresses {
			serverAddress, password := addr[0], addr[1]
			if !strings.HasPrefix(serverAddress, "http://") && !strings.HasPrefix(serverAddress, "https://") {
				serverAddress = "http://" + serverAddress
			}
			serverURL := serverAddress + "/upload"
			err = uploadFile(serverURL, password, username, filename, hideResponse)
			if err != nil {
				fmt.Printf("\nError uploading file to %s: %v\n", serverAddress, err)
			}
		}
	} else {
		args := flag.Args()
		if len(args) != 2 {
			fmt.Println("Usage: oc [-u username] [-d datafile] [-h hide server response] \n          -f <filename> <server_address:port> <password>")
			os.Exit(1)
		}
		serverAddress, password := args[0], args[1]
		if !strings.HasPrefix(serverAddress, "http://") && !strings.HasPrefix(serverAddress, "https://") {
			serverAddress = "http://" + serverAddress
		}
		serverURL := serverAddress + "/upload"
		err = uploadFile(serverURL, password, username, filename, hideResponse)
		if err != nil {
			fmt.Printf("\nError uploading file: %v\n", err)
			os.Exit(1)
		}
	}
}

func readDataFile(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var addresses [][]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by whitespace and filter out empty strings
		parts := strings.Fields(line)
		
		// Ensure we have exactly two non-empty parts
		var validParts []string
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				validParts = append(validParts, trimmed)
			}
		}

		if len(validParts) != 2 {
			continue // Skip invalid lines instead of returning error
		}

		addresses = append(addresses, validParts)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no valid entries found in data file")
	}

	return addresses, nil
}

func uploadFile(serverURL, password, username, filename string, hideResponse bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)

	startTime = time.Now()
	go func() {
		defer pipeWriter.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			fmt.Printf("Error creating form file: %v\n", err)
			return
		}

		_, err = io.Copy(part, file)
		if err != nil {
			fmt.Printf("Error copying file content: %v\n", err)
		}
	}()

	dialer, err := proxy.SOCKS5("tcp", "localhost:9050", nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("can't connect to the Tor proxy: %v", err)
	}
	httpTransport := &http.Transport{Dial: dialer.Dial}
	client := &http.Client{Transport: httpTransport}
	fmt.Println("Using Tor network")

	request, err := http.NewRequest("POST", serverURL, pipeReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Password", password)
	if username != "" {
		request.Header.Set("X-Username", username)
	}

	fmt.Print("Send file...\n")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", response.Status, string(bodyBytes))
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("\nFile sent successfully. Total time: %s\n\n", formatDuration(elapsedTime))

	if !hideResponse {
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		fmt.Println("Onion Courier Response:", string(responseBody))
	}

	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	
	if h > 0 {
		return fmt.Sprintf("%d hours %d minutes %d seconds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%d minutes %d seconds", m, s)
	} else {
		return fmt.Sprintf("%d seconds", s)
	}
}