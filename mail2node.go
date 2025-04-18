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
	"os"
	"strings"

	"github.com/awnumar/memguard"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/net/proxy"
	"mime/multipart"
	"net/http"
)

const maxFileSize = 4096 * 1024

var privateKeyPath string
var privateKeyLocked *memguard.LockedBuffer

func main() {
	flag.StringVar(&privateKeyPath, "s", "", "Path to the private key file")
	flag.Parse()

	if privateKeyPath == "" {
		fmt.Fprintln(os.Stderr, "Error: please provide the private key file path using -s")
		os.Exit(1)
	}

	var err error
	privateKeyLocked, err = loadPEM(privateKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading private key: %v\n", err)
		os.Exit(1)
	}
	defer privateKeyLocked.Destroy()

	input, err := io.ReadAll(io.LimitReader(os.Stdin, maxFileSize))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	decryptedContent, err := decryptContent(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting: %v\n", err)
		os.Exit(1)
	}
	defer decryptedContent.Destroy()

	lines := strings.Split(string(decryptedContent.Bytes()), "\n")
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
		fmt.Fprintln(os.Stderr, "Error: no valid headers found")
		os.Exit(1)
	}

	headerParts := strings.SplitN(headers[0], " ", 3)
	if len(headerParts) != 3 || headerParts[0] != "X-OC-To:" {
		fmt.Fprintln(os.Stderr, "Error: invalid header format (missing X-OC-To)")
		os.Exit(1)
	}

	onionAddress := headerParts[1]
	password := strings.TrimSpace(headerParts[2])
	newMessage := strings.Join(append(headers[1:], "", messageBody), "\n")

	response, err := sendToOnionAddress([]byte(newMessage), onionAddress, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Message sent successfully.\nResponse from onion service:\n%s\n", response)
}

func sendToOnionAddress(message []byte, onionAddress, password string) (string, error) {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("SOCKS5 error: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Dial: dialer.Dial},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "message.txt")
	if err != nil {
		return "", err
	}

	_, err = part.Write(message)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("http://%s/upload", onionAddress)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Password", password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response: %d - %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

func decryptContent(content []byte) (*memguard.LockedBuffer, error) {
	reader := bytes.NewReader(content)
	var writer bytes.Buffer
	if err := decrypt(privateKeyLocked, reader, &writer); err != nil {
		return nil, err
	}
	return memguard.NewBufferFromBytes(writer.Bytes()), nil
}

func loadPEM(filename string) (*memguard.LockedBuffer, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("PEM decoding failed")
	}
	return memguard.NewBufferFromBytes(block.Bytes), nil
}

func ed25519PrivateKeyToCurve25519(pk ed25519.PrivateKey) []byte {
	h := sha512.New()
	h.Write(pk.Seed())
	out := h.Sum(nil)
	return out[:curve25519.ScalarSize]
}

func decrypt(privKey *memguard.LockedBuffer, reader io.Reader, writer io.Writer) error {
	privKeyBytes := privKey.Bytes()
	ed25519PrivKey := ed25519.PrivateKey(privKeyBytes)
	curve25519PrivKey := ed25519PrivateKeyToCurve25519(ed25519PrivKey)

	locked := memguard.NewBufferFromBytes(curve25519PrivKey)
	defer locked.Destroy()

	encoded, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return err
	}
	if len(decoded) < 56 {
		return errors.New("encoded input too short")
	}

	ephemeralPub := decoded[:32]
	nonce := decoded[32:56]
	ciphertext := decoded[56:]

	sharedSecret, err := curve25519.X25519(locked.Bytes(), ephemeralPub)
	if err != nil {
		return err
	}

	secret := memguard.NewBufferFromBytes(sharedSecret)
	defer secret.Destroy()

	aead, err := chacha20poly1305.NewX(secret.Bytes())
	if err != nil {
		return err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	_, err = writer.Write(plaintext)
	return err
}
