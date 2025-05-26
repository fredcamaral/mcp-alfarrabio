// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test.go [server|client]")
		return
	}
	
	switch os.Args[1] {
	case "server":
		// Run as server
		main()
	case "client":
		// Run test client
		RunClientTest()
	case "both":
		// Start server in background
		fmt.Println("Starting server...")
		cmd := exec.Command("go", "run", "main.go")
		cmd.Env = append(os.Environ(), "MCP_PORT=3000")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			return
		}
		
		// Wait for server to start
		fmt.Println("Waiting for server to start...")
		time.Sleep(2 * time.Second)
		
		// Run client tests
		RunClientTest()
		
		// Kill server
		cmd.Process.Kill()
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}