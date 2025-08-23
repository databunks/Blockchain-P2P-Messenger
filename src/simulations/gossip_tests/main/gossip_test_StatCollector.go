package main

import (
	"blockchain-p2p-messenger/src/peerDetails"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)


var reachabilityCount = 0
var lastReceivedMessageTime time.Time
var timestamp_arrived []uint64

func main() {
	
	
    go ListenOnPort(":3002")

	// fmt.Println("Starting timer in 20s.....")
	// time.Sleep(time.Second * 20)

	var start time.Time = time.Now()
	// SendMessage("Start!", "room-xyz-987", 3002)
	fmt.Println("Timer Started!!")

	time.Sleep(time.Minute * 1)
	
	fmt.Println("Reachability Count: %d", reachabilityCount)
	fmt.Println("Latency: %s", lastReceivedMessageTime.UnixMilli()) 

}


func ListenOnPort(port string) {
	// Listen on the specified port
    listener, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
    defer listener.Close()

    fmt.Printf("Server is listening on port %s...\n", port)

    // Accept incoming connections
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error accepting connection: %v", err)
            continue
        }
        fmt.Printf("New connection established: %v\n", conn.RemoteAddr())

        // Handle connection 
        go handleConnection(conn)
    }

}


func handleConnection(conn net.Conn) {
	// Close connection
    defer conn.Close()


    for {
        defer conn.Close()

		
	// Read message sent by client
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	// Unmarshal message (you JSON-encoded it before sending)
	var msg string
	err = json.Unmarshal(buffer[:n], &msg)
	if err != nil {
		fmt.Println("Error unmarshaling message:", err)
		return
	}

	fmt.Println("Received from peer:", msg)

		var filteredMessage string = msg

		if strings.HasPrefix("Message Reached To Peer ", msg){
			filteredMessage = strings.Split(msg, "Message Reached To Peer ")[0]
		}
		

		switch filteredMessage {

			// Implementation
			case "Message Reached To Peer ":
				reachabilityCount++
				break


			// Control
			case "Received Censored Message":
				reachabilityCount++
				str := strings.Split(filteredMessage, " ")[1]
				ts, err := strconv.ParseUint(str, 10, 64)

				if (err != nil){
					fmt.Println("Error parsing uint " + err.Error())
				}

				timestamp_arrived = append(timestamp_arrived, ts) 
				
				
				break
		}

        // Response back to client
        _, err = conn.Write([]byte("Message received!\n"))
        if err != nil {
            fmt.Printf("Error writing to client: %v\n", err)
        }
    }
}


func SendMessage(messageContent string, roomID string, port uint64) error {
	peers := peerDetails.GetPeersInRoom(roomID)



	var wg sync.WaitGroup

	for _, peer := range peers {
		// ignore if peer in blockchain

		wg.Add(1)
		go func(peer peerDetails.Peer) {
			defer wg.Done()

			// Dial peer
			fmt.Printf("Establishing connection with %s, %s.......\n", peer.IP, peer.PublicKey)
			address := net.JoinHostPort(peer.IP, fmt.Sprintf("%d", port))
			fmt.Println(address)

			conn, err := net.Dial("tcp", address)
			if err != nil {
				log.Printf("Error connecting to %s: %v\n", peer.IP, err)
				return
			}
			defer conn.Close()

			// Marshal message
			msgBytes, err := json.Marshal(messageContent)
			if err != nil {
				log.Printf("Error marshaling message for %s: %v\n", peer.IP, err)
				return
			}

			// Send message
			_, err = conn.Write(msgBytes)
			if err != nil {
				log.Printf("Error sending message to %s: %v\n", peer.IP, err)
				return
			}

			// Read the response from the peer
			buffer := make([]byte, 1024) // Buffer to store incoming data
			n, err := conn.Read(buffer)
			if err != nil {
				log.Printf("Error reading response from %s: %v\n", peer.IP, err)
				return
			}

			// Print the response received from the peer
			response := string(buffer[:n])
			fmt.Printf("Response from %s: %s\n", peer.IP, response)

		}(peer) // Pass peer as an argument to avoid closure capture issues
	}

	wg.Wait() // Wait for all goroutines to finish
	return nil
}