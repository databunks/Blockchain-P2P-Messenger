package main

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// VM1 Configuration
var publicKey_VM1 string = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var publicKey2_VM2 string = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 string = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var publicKey4_VM4 string = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
var isAdmin bool = true
var roomID string = "room-xyz-987"

// Test state
var isTestRunning bool = false
var testMutex sync.Mutex
var messageCount int = 0
var messageCountMutex sync.Mutex

func main() {
	fmt.Println("=== GOSSIP TEST VM1 STARTED ===")
	fmt.Printf("Public Key: %s\n", publicKey_VM1[:16]+"...")
	fmt.Printf("Room ID: %s\n", roomID)
	fmt.Printf("Is Admin: %t\n", isAdmin)

	// Initialize peers
	initializePeers()

	// Start command listener
	go startCommandListener(3000)

	// Start network
	fmt.Println("Initializing network...")
	network.InitializeNetwork(roomID, true, true)

	// Keep the program running
	select {}
}

// initializePeers adds all peers to the room
func initializePeers() {
	fmt.Println("Initializing peers...")

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), isAdmin, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	fmt.Printf("✅ Added %d peers to room %s\n", 4, roomID)
}

// startCommandListener listens for commands from the stat collector
func startCommandListener(port int) {
	var yggdrasilNodeInfo = network.GetYggdrasilNodeInfo()
	address := fmt.Sprintf("[%s]:%d", yggdrasilNodeInfo.Address, port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	fmt.Printf("🎧 Command listener started on %s\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		go handleCommandConnection(conn)
	}
}

// handleCommandConnection handles incoming command connections
func handleCommandConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error reading command: %v", err)
		}
		return
	}

	command := strings.TrimSpace(string(buffer[:n]))
	fmt.Printf("📨 Received command: %s\n", command)

	// Process the command
	processCommand(command)

	// Send acknowledgment
	response := "Command received and processed"
	conn.Write([]byte(response))
}

// processCommand processes incoming commands
func processCommand(command string) {
	switch command {
	case "START_GOSSIP_TEST":
		startGossipTest()
	case "CLEAR_MESSAGES":
		clearMessages()
	case "STOP_TEST":
		stopTest()
	default:
		fmt.Printf("⚠️  Unknown command: %s\n", command)
	}
}

// startGossipTest starts the gossip test
func startGossipTest() {
	testMutex.Lock()
	defer testMutex.Unlock()

	if isTestRunning {
		fmt.Println("⚠️  Test already running, ignoring start command")
		return
	}

	isTestRunning = true
	messageCountMutex.Lock()
	messageCount = 0
	messageCountMutex.Unlock()

	fmt.Println("🚀 Starting gossip test...")

	// Send test start notification to stat collector
	sendToStatCollector("GOSSIP_START")

	// Wait a moment for network to stabilize
	time.Sleep(2 * time.Second)

	// Send the censored message
	sendCensoredMessage()
}

// sendCensoredMessage sends the censored message to the network
func sendCensoredMessage() {
	fmt.Println("📤 Sending censored message...")

	// Initialize gossip network
	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, true, false, false)
	if err != nil {
		fmt.Printf("❌ Failed to initialize gossip network: %v\n", err)
		return
	}

	// Send the censored message
	messageContent := "This is a censored message that should reach all nodes!"
	messageType := "chat"
	messageID := fmt.Sprintf("censored_%d", time.Now().UnixNano())

	fmt.Printf("📝 Sending message: %s\n", messageContent)

	// Send via gossip network
	gossipNet.GossipMessage(messageType, "broadcast", messageContent, 0, roomID, messageID)

	// Track message overhead
	messageCountMutex.Lock()
	messageCount++
	messageCountMutex.Unlock()

	// Send message sent notification to stat collector
	sendToStatCollector("Message Sent")

	fmt.Println("✅ Censored message sent successfully")

	// Wait for message propagation
	time.Sleep(5 * time.Second)

	// Send completion notification
	sendToStatCollector("Message Reached To Peer VM1")
}

// clearMessages clears message tracking
func clearMessages() {
	fmt.Println("🧹 Clearing message tracking...")

	messageCountMutex.Lock()
	messageCount = 0
	messageCountMutex.Unlock()

	testMutex.Lock()
	isTestRunning = false
	testMutex.Unlock()

	fmt.Println("✅ Message tracking cleared")
}

// stopTest stops the current test
func stopTest() {
	fmt.Println("🛑 Stopping test...")

	testMutex.Lock()
	isTestRunning = false
	testMutex.Unlock()

	fmt.Println("✅ Test stopped")
}

// sendToStatCollector sends a message to the stat collector
func sendToStatCollector(message string) {
	statCollectorPublicKey := "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"

	peers := peerDetails.GetPeersInRoom(roomID)
	for _, peer := range peers {
		if peer.PublicKey == statCollectorPublicKey {
			fmt.Printf("📊 Sending to stat collector: %s\n", message)

			// Send message to stat collector
			err := sendMessageToNode(peer.IP, 3002, message)
			if err != nil {
				fmt.Printf("❌ Failed to send to stat collector: %v\n", err)
			} else {
				fmt.Printf("✅ Sent to stat collector: %s\n", message)
			}
			break
		}
	}
}

// sendMessageToNode sends a message to a specific node
func sendMessageToNode(nodeIP string, port int, message string) error {
	address := net.JoinHostPort(nodeIP, fmt.Sprintf("%d", port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", address, err)
	}
	defer conn.Close()

	// Marshal message as JSON
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Send message
	_, err = conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	response := string(buffer[:n])
	fmt.Printf("📨 Response from %s: %s\n", address, response)

	return nil
}

// handleIncomingMessage handles incoming messages from other nodes
func handleIncomingMessage(message string, senderIP string) {
	fmt.Printf("📨 Received message from %s: %s\n", senderIP, message)

	// Check if this is a censored message
	if strings.Contains(message, "censored") || strings.Contains(message, "This is a censored message") {
		fmt.Println("🎯 Received censored message!")

		// Send receipt notification to stat collector
		sendToStatCollector("Received Censored Message")

		// Forward the message to other nodes (gossip behavior)
		forwardMessage(message)
	}
}

// forwardMessage forwards a message to other nodes
func forwardMessage(message string) {
	fmt.Println("🔄 Forwarding message to other nodes...")

	peers := peerDetails.GetPeersInRoom(roomID)
	for _, peer := range peers {
		if peer.PublicKey != publicKey_VM1 { // Don't send back to self
			fmt.Printf("📤 Forwarding to %s...\n", peer.IP)

			err := sendMessageToNode(peer.IP, 3000, message)
			if err != nil {
				fmt.Printf("❌ Failed to forward to %s: %v\n", peer.IP, err)
			} else {
				// Track message overhead
				messageCountMutex.Lock()
				messageCount++
				messageCountMutex.Unlock()

				// Send message sent notification
				sendToStatCollector("Message Sent")
			}
		}
	}
}

// GetTestStatus returns the current test status
func GetTestStatus() (bool, int) {
	testMutex.Lock()
	defer testMutex.Unlock()

	messageCountMutex.Lock()
	defer messageCountMutex.Unlock()

	return isTestRunning, messageCount
}

// GetMessageCount returns the current message count
func GetMessageCount() int {
	messageCountMutex.Lock()
	defer messageCountMutex.Unlock()
	return messageCount
}
