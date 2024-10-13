package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/net/proxy"
)

const (
	serverPassword = "secretPassword" // Set your desired server password here
	maxFileSize    = 28 * 1024        // 28 KB in bytes
)

var privateKeyPath string

func main() {
	flag.StringVar(&privateKeyPath, "s", "", "Path to the private key file")
	flag.Parse()

	if privateKeyPath == "" {
		log.Fatal("Please provide the path to the private key file using the -s flag")
	}

	http.HandleFunc("/upload", handleUpload)
	fmt.Println("Server is running on http://localhost:8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Password check
	if r.Header.Get("X-Password") != serverPassword {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Limit the request body size to maxFileSize
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	err := r.ParseMultipartForm(maxFileSize)
	if err != nil {
		http.Error(w, "File too large or error parsing form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file content", http.StatusInternalServerError)
		return
	}

	if len(content) > maxFileSize {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// Decrypt the content
	decryptedContent, err := decryptContent(content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decrypting content: %v", err), http.StatusInternalServerError)
		return
	}

	lines := strings.Split(string(decryptedContent), "\n")
	var headers []string
	var messageBody string

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			messageBody = strings.Join(lines[i+1:], "\n")
			break
		}
		headers = append(headers, line)
	}

	if len(headers) == 0 {
		http.Error(w, "No valid headers found", http.StatusBadRequest)
		return
	}

	// Processing the first header
	headerParts := strings.SplitN(headers[0], " ", 3)
	if len(headerParts) != 3 || headerParts[0] != "X-OC-To:" {
		http.Error(w, "Invalid header format", http.StatusBadRequest)
		return
	}

	onionAddress := headerParts[1]
	password := strings.TrimSpace(headerParts[2])

	// Create new message without first header
	newMessage := strings.Join(append(headers[1:], "", messageBody), "\n")

	response, err := sendToOnionAddress([]byte(newMessage), onionAddress, password)
	if err != nil {
		log.Printf("Error sending to onion address: %v", err)
		http.Error(w, fmt.Sprintf("Error sending to onion address: %v", err), http.StatusInternalServerError)
		return
	}

	guardOnionURL := "w7t3g7oo5naebqwlezshgkgczttjn7x3re3farrzwa6bttvbnm5fcsad.onion:8084" // Replace with your actual Guard Onion URL.
	nickname := "Bobby" // Replace with your actual server's nickname

	responseMsg := "\n============================\n"
	responseMsg += "File received, decrypted and sent by:\n%s\n"
	responseMsg += "No data is stored or logged by Onion Courier Guard %s.\n\n"
	responseMsg += "Target Onion Courier Response:\n%s\n"
	responseMsg += "============================\n"
	responseMsg += "%s"

	fmt.Fprintf(w, responseMsg, guardOnionURL, nickname, onionAddress, response)
}

func sendToOnionAddress(message []byte, onionAddress, password string) (string, error) {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("error creating SOCKS5 dialer: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "message.txt")
	if err != nil {
		return "", fmt.Errorf("error creating form file: %v", err)
	}

	_, err = part.Write(message)
	if err != nil {
		return "", fmt.Errorf("error writing message to form file: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("error closing multipart writer: %v", err)
	}

	url := fmt.Sprintf("http://%s/upload", onionAddress)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Password", password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-OK status: %d - %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// Decryption functions from minicrypt.go

func decryptContent(content []byte) ([]byte, error) {
	privKeyBytes, err := loadPEM(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading private key: %v", err)
	}
	privKey := ed25519.PrivateKey(privKeyBytes)

	reader := bytes.NewReader(content)
	var writer bytes.Buffer
	err = decrypt(privKey, reader, &writer)
	if err != nil {
		return nil, fmt.Errorf("error decrypting: %v", err)
	}

	return writer.Bytes(), nil
}

func loadPEM(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	pemData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("PEM decoding failed")
	}
	return block.Bytes, nil
}

func ed25519PrivateKeyToCurve25519(pk ed25519.PrivateKey) []byte {
	h := sha512.New()
	h.Write(pk.Seed())
	out := h.Sum(nil)
	return out[:curve25519.ScalarSize]
}

func decrypt(privKey ed25519.PrivateKey, reader io.Reader, writer io.Writer) error {
	curve25519PrivKey := ed25519PrivateKeyToCurve25519(privKey)

	// Read and decode the input (base64)
	encodedInput, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	decodedInput, err := base64.StdEncoding.DecodeString(string(encodedInput))
	if err != nil {
		return err
	}

	// Extract the ephemeral public key, nonce, and ciphertext
	curve25519EphemeralPubKey := decodedInput[:32]
	nonce := decodedInput[32:56]
	ciphertext := decodedInput[56:]

	// Perform X25519 key exchange
	sharedSecret, err := curve25519.X25519(curve25519PrivKey, curve25519EphemeralPubKey)
	if err != nil {
		return err
	}

	// XChaCha20-Poly1305 setup
	aead, err := chacha20poly1305.NewX(sharedSecret)
	if err != nil {
		return err
	}

	// Decrypt the data
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// Write the decrypted data
	_, err = writer.Write(plaintext)
	return err
}

