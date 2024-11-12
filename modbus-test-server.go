package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	// Create a TCP listener
	listener, err := net.Listen("tcp", "0.0.0.0:502")
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()
	fmt.Println("Modbus test server listening on port 502")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		data := make([]byte, 1024)
		n, err := conn.Read(data)
		if err != nil {
			// If the error is EOF, the connection was closed by the client, which is expected.
			if err.Error() == "EOF" {
				log.Println("Client disconnected.")
				break
			} else {
				log.Printf("Error reading data: %v", err)
				break
			}
		}
		fmt.Printf("Received data: %x\n", data[:n])

		// Process the received data
		response := processData(data[:n])

		// Echo the processed data back to the client
		_, err = conn.Write(response)
		if err != nil {
			log.Printf("Error writing data: %v", err)
			return
		}
	}
}

func processData(data []byte) []byte {
	// Implement your data processing logic here
	// For demonstration, let's just return the received data as-is
	return data
}
