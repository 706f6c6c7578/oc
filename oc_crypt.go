package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/curve25519"
	"crypto/sha512"
	"filippo.io/edwards25519"
)

// Write PEM files
func savePEM(filename string, data []byte, pemType string) error {
	block := &pem.Block{
		Type:  pemType,
		Bytes: data,
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

// Load PEM files
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

// Ed25519 to Curve25519 conversions
func ed25519PrivateKeyToCurve25519(pk ed25519.PrivateKey) []byte {
	h := sha512.New()
	h.Write(pk.Seed())
	out := h.Sum(nil)
	return out[:curve25519.ScalarSize]
}

func ed25519PublicKeyToCurve25519(pk ed25519.PublicKey) ([]byte, error) {
	p, err := new(edwards25519.Point).SetBytes(pk)
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func generateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return priv, pub, err
}

func sharedSecret(pub, priv []byte) ([]byte, error) {
	xPriv := ed25519PrivateKeyToCurve25519(ed25519.PrivateKey(priv))
	xPub, err := ed25519PublicKeyToCurve25519(pub)
	if err != nil {
		return nil, err
	}
	return curve25519.X25519(xPriv, xPub)
}

// Encryption function
func encrypt(pubKey []byte, reader io.Reader, writer io.Writer) error {
	// Simulated encryption (this part should be adapted)
	var inputData []byte
	buffer := bufio.NewReader(reader)

	for {
		line, err := buffer.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		inputData = append(inputData, []byte(line)...)
	}

	// Simulated encryption (base64 encode the entire input data)
	encoded := base64.StdEncoding.EncodeToString(inputData)
	_, err := writer.Write([]byte(chunk64(encoded) + "\r\n"))
	return err
}

// Decryption function
func decrypt(privKey []byte, reader io.Reader, writer io.Writer) error {
	// Simulated decryption (this part should be adapted)
	buffer := bufio.NewReader(reader)
	var inputData string
	for {
		line, err := buffer.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		inputData += strings.TrimSpace(line)
	}
	decoded, err := base64.StdEncoding.DecodeString(inputData)
	if err != nil {
		return err
	}
	_, err = writer.Write(decoded)
	return err
}

// Helper function to wrap Base64 output at 64 characters
func chunk64(input string) string {
	var output strings.Builder
	for len(input) > 64 {
		output.WriteString(input[:64] + "\r\n")
		input = input[64:]
	}
	output.WriteString(input)
	return output.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: oc_crypt public.pem < infile > outfile")
		fmt.Println("       oc_crypt -d private.pem < infile > outfile")
		fmt.Println("       oc_crypt -g generate key pair and save it")
		os.Exit(1)
	}

	if os.Args[1] == "-g" {
		// Generate key pair and save it
		priv, pub, err := generateKeyPair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating key pair: %v\n", err)
			os.Exit(1)
		}

		err = savePEM("public.pem", pub, "PUBLIC KEY")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving public.pem: %v\n", err)
			os.Exit(1)
		}

		err = savePEM("private.pem", priv, "PRIVATE KEY")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving private.pem: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Key pair successfully generated.")
		os.Exit(0)
	}

	if os.Args[1] == "-d" {
		// Decrypt
		privKey, err := loadPEM(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading private key: %v\n", err)
			os.Exit(1)
		}
		err = decrypt(privKey, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Encrypt
		pubKey, err := loadPEM(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading public key: %v\n", err)
			os.Exit(1)
		}
		err = encrypt(pubKey, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting: %v\n", err)
			os.Exit(1)
		}
	}
}
