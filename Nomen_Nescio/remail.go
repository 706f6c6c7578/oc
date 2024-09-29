package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	REMAILER_EMAIL = "remailer@dizum.com"
)

func main() {
	info, err := os.Stdin.Stat()
	if err != nil {
		fmt.Println("Error reading stdin:", err)
		printUsage()
		os.Exit(1)
	}

	if info.Mode()&os.ModeCharDevice != 0 || info.Size() <= 0 {
		fmt.Println("No input provided.")
		printUsage()
		os.Exit(1)
	}

	if err := encryptAndPrintMessage(REMAILER_EMAIL); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func encryptAndPrintMessage(TO string) error {
	reader := bufio.NewReader(os.Stdin)
	var messageBuilder strings.Builder
	
	// Lese die Eingabenachricht
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading input: %v", err)
		}
		messageBuilder.WriteString(line)
	}
	MSG := messageBuilder.String()

	// Überprüfe, ob die Nachricht bereits mit '::' beginnt
	if !strings.HasPrefix(MSG, "::") {
		MSG = fmt.Sprintf("::\nAnon-To: %s\n\n%s", TO, MSG)
	}

	cmd := exec.Command("gpg", "-ea", "--batch", "--trust-model", "always", "-r", TO)
	cmd.Stdin = strings.NewReader(MSG)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running GPG: %v", err)
	}

	ENCMSG := out.String()

	fmt.Printf("::\nEncrypted: PGP\n\n%s", ENCMSG)
	return nil
}

func printUsage() {
	fmt.Printf(`Usage: %s < input_file

Reads a message from standard input, encrypts it for the hardcoded remailer (%s),
adds necessary headers, and prints the result to standard output.

Example:
  $ echo "Secret message" | %s > encrypted_message.txt

`, os.Args[0], REMAILER_EMAIL, os.Args[0])
}
