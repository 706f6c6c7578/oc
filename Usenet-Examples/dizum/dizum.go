package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "runtime"
    "time"
)

const password = "secretPassword" // Set your desired password here

func main() {
    http.HandleFunc("/upload", handleUpload)
    fmt.Println("Server is running on http://localhost:8081")
    http.ListenAndServe(":8081", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
        return
    }

    // Check the password
    if r.Header.Get("X-Password") != password {
        http.Error(w, "Invalid password", http.StatusUnauthorized)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Create a pipe to pass the data to the external program
    reader, writer := io.Pipe()
    defer reader.Close()
    defer writer.Close()

    // Write the file content to the pipe
    go func() {
        _, err := io.Copy(writer, file)
        if err != nil {
            log.Println("Error writing to pipe:", err)
        }
        writer.Close()
    }()

    // Execute the external program with the pipe as stdin
    var cmd *exec.Cmd
    if runtime.GOOS == "windows" {
        cmd = exec.Command("cmd", "/C", "D:\\tools\\m2n.exe")
    } else {
        cmd = exec.Command("sh", "-c", "/usr/local/bin/m2n")
    }
    cmd.Stdin = reader

    // Set the output to stdout and stderr
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Debugging output
    fmt.Fprintf(os.Stderr, "Executing command: %s\n", cmd.String())

    if err := cmd.Run(); err != nil {
        log.Println("Error running command:", err)
    }

    // Output to stderr with timestamp and username (if provided)
    currentTime := time.Now().Format("15:04:05")
    username := r.Header.Get("X-Username")
    if username == "" {
        username = "Anonymous"
    }
    fmt.Fprintf(os.Stderr, "File %s received at %s by %s\n", header.Filename, currentTime, username)

    // Output to the client
    fmt.Fprintf(w, "File %s received!", header.Filename)
}
