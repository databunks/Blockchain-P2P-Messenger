package gossipnetwork

import (
	"blockchain-p2p-messenger/src/blockchain"
	"blockchain-p2p-messenger/src/consensus"
	"blockchain-p2p-messenger/src/network"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Gossip Protocol Types
type GossipMessage struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Data         string `json:"data"`
	Sender       string `json:"sender"`
	RoomID       string `json:"room_id"`
	Timestamp    int64  `json:"timestamp"`
	TTL          int    `json:"ttl"`
	PublicKey    string `json:"public_key"`
	Signature    string `json:"signature"`
	Category     string `json:"category"` // Add missing fields
	TargetID     string `json:"target_id"`
	AcksReceived int    `json:"acks_received"`
}

type GossipNode struct {
	ID        uint64
	Address   string
	PublicKey string // Add PublicKey field
	LastSeen  time.Time
	IsAlive   bool
}

// Yggdrasil peer types
type Peer struct {
	Remote    string  `json:"remote"`
	Up        bool    `json:"up"`
	Inbound   bool    `json:"inbound"`
	Address   string  `json:"address"`
	Key       string  `json:"key"`
	Port      int     `json:"port"`
	Priority  int     `json:"priority"`
	Cost      int     `json:"cost"`
	BytesRecv float64 `json:"bytes_recvd"`
	BytesSent float64 `json:"bytes_sent"`
	Uptime    float64 `json:"uptime"`
	Latency   float64 `json:"latency"`
}

type PeerList struct {
	Peers []Peer `json:"peers"`
}

type YggdrasilNodeInfo struct {
	BuildName      string `json:"build_name"`
	BuildVersion   string `json:"build_version"`
	Key            string `json:"key"`
	Address        string `json:"address"`
	RoutingEntries int    `json:"routing_entries"`
	Subnet         string `json:"subnet"`
}

// GossipNetwork integrates gossip protocol with Yggdrasil peer management
type GossipNetwork struct {
	nodeID         uint64
	gossipPeers    map[uint64]*GossipNode
	messageHistory map[string]bool // Track seen messages
	gossipMutex    sync.RWMutex
	roomID         string
	port           uint64

	// Network connection management
	conns     map[string]net.Conn
	connMutex sync.RWMutex
	listener  net.Listener

	// Authentication
	disableAuth bool
	privateKey  ed25519.PrivateKey
	publicKey   ed25519.PublicKey

	// Message processing
	msgsToProcess []GossipMessage
	processedAcks map[string]map[string]bool // messageID -> peerPublicKey -> bool
}

var nodes []*consensus.Server

var peerIDs []uint64

var nodeIDs []uint64

var msgsToProcess []GossipMessage

var isCensorshipAttackerNode bool
var blockChainState bool

var thresholdAcks int

func init() {
	gob.Register(&consensus.Transaction{})
}

// NewGossipNetwork creates a new integrated gossip network
func NewGossipNetwork(nodeID uint64, roomID string, port uint64) *GossipNetwork {
	// Load private key for authentication
	privateKey, publicKey := loadKeys()

	return &GossipNetwork{
		nodeID:         nodeID,
		roomID:         roomID,
		port:           port,
		gossipPeers:    make(map[uint64]*GossipNode),
		messageHistory: make(map[string]bool),
		conns:          make(map[string]net.Conn),
		privateKey:     privateKey,
		publicKey:      publicKey,
		disableAuth:    false, // Set to true to disable authentication
		msgsToProcess:  make([]GossipMessage, 0),
		processedAcks:  make(map[string]map[string]bool),
	}
}

// loadKeys loads the private and public keys from keydetails.env
func loadKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get the ED25519 private key from the environment variable
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("PRIVATE_KEY is not set in keydetails.env file")
	}

	// Convert the hex string to bytes
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatal("Failed to decode private key:", err)
	}

	// Ensure the private key is the correct length (32 bytes for ED25519)
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		log.Fatal("Invalid private key size, expected 32 bytes")
	}

	// Generate the public key from the private key
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return privateKey, publicKey
}

// SignMessage signs a message with the private key and returns the signature
func (gn *GossipNetwork) SignMessage(message []byte) []byte {
	signature := ed25519.Sign(gn.privateKey, message)
	return signature
}

func (gn *GossipNetwork) VerifyMessageSignature(messageContent []byte, signatureHex string, publicKeyHex string) bool {
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false
	}

	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(publicKeyBytes), messageContent, signatureBytes)
}

func (gn *GossipNetwork) authenticateMessage(msg GossipMessage, peer GossipNode) bool {
	if gn.disableAuth {
		return true
	}

	if !gn.isInRoom(peer.PublicKey) {
		return false
	}

	// Verify message signature
	messageBytes := []byte(fmt.Sprintf("%s%s%s%d", msg.ID, msg.Type, msg.Data, msg.Timestamp))

	// Try multiple signature formats for compatibility
	formats := [][]byte{
		messageBytes,     // Current format
		[]byte(msg.Data), // Legacy format
		[]byte(fmt.Sprintf("%s%s%s", msg.ID, msg.Type, msg.Data)),                             // Alternative format
		[]byte(fmt.Sprintf("%s%s%d", msg.Type, msg.Data, msg.Timestamp)),                      // Timestamp format
		[]byte(fmt.Sprintf("%s%s%s%d%d", msg.ID, msg.Type, msg.Data, msg.Timestamp, msg.TTL)), // Old TTL format
	}

	for _, format := range formats {
		if gn.VerifyMessageSignature(format, msg.Signature, peer.PublicKey) {
			break
		}
	}

	if !gn.VerifyMessageSignature(messageBytes, msg.Signature, peer.PublicKey) {
		// Try fallback formats
		verified := false
		for i := 1; i < len(formats); i++ {
			if gn.VerifyMessageSignature(formats[i], msg.Signature, peer.PublicKey) {
				verified = true
				break
			}
		}
		if !verified {
			fmt.Printf("Authentication failed: invalid signature from %s\n", peer.PublicKey)
			return false
		}
	}

	// Check if sender's public key matches peer's public key
	if msg.PublicKey != peer.PublicKey {
		fmt.Printf("Authentication failed: public key mismatch. Message: %s, Peer: %s\n", msg.PublicKey, peer.PublicKey)
		return false
	}

	// Check if message is for the correct room
	if msg.RoomID != gn.roomID {
		fmt.Printf("Authentication failed: wrong room. Message room: %s, Current room: %s\n", msg.RoomID, gn.roomID)
		return false
	}

	fmt.Printf("Authentication successful for message from %s\n", peer.PublicKey)
	return true
}

// InitializeGossipNetwork sets up the complete gossip network
func InitializeGossipNetwork(roomID string, port uint64, toggleAttacker bool, toggleBlockchain bool) (*GossipNetwork, error) {
	peerCount := len(peerDetails.GetPeersInRoom(roomID))
	thresholdAcks = int(float64(peerCount) * 0.66) // 66% of peers
	if thresholdAcks < 2 {
		thresholdAcks = 2 // Minimum threshold of 2 acks
	}
	isCensorshipAttackerNode = toggleAttacker
	blockChainState = toggleBlockchain

	fmt.Println("Initializing integrated gossip network...")
	fmt.Println(PublicKeyToID(GetYggdrasilNodeInfo().Key))

	// Create gossip network instance
	gossipNet := NewGossipNetwork(PublicKeyToID(GetYggdrasilNodeInfo().Key), roomID, port)

	// Get Yggdrasil peers
	yggdrasilPeers, err := gossipNet.GetYggdrasilPeers("unique")
	if err != nil {
		return nil, fmt.Errorf("failed to get Yggdrasil peers: %v", err)
	}

	fmt.Printf("Found %d Yggdrasil peers for gossip network\n", len(yggdrasilPeers))

	// Add Yggdrasil peers to gossip network
	for _, yggPeer := range yggdrasilPeers {
		// Convert Yggdrasil peer key to uint64 ID
		peerID := gossipNet.PublicKeyToID(yggPeer.Key)

		// Add to gossip network
		gossipNet.gossipPeers[peerID] = &GossipNode{
			ID:        peerID,
			Address:   yggPeer.Address, // Use Yggdrasil IP address
			PublicKey: yggPeer.Key,     // Fix: Set the PublicKey field
			LastSeen:  time.Now(),
			IsAlive:   true,
		}

		fmt.Printf("Added Yggdrasil peer to gossip network: ID=%d, Address=%s, Key=%s\n",
			peerID, yggPeer.Address, yggPeer.Key)
	}

	// Add current node to gossip peers (so it can authenticate its own messages)
	currentNodeInfo := GetYggdrasilNodeInfo()
	currentNodeID := gossipNet.PublicKeyToID(currentNodeInfo.Key)

	// Only add if not already present
	if _, exists := gossipNet.gossipPeers[currentNodeID]; !exists {
		gossipNet.gossipPeers[currentNodeID] = &GossipNode{
			ID:        currentNodeID,
			Address:   currentNodeInfo.Address,
			PublicKey: currentNodeInfo.Key,
			LastSeen:  time.Now(),
			IsAlive:   true,
		}
		fmt.Printf("Added current node to gossip network: ID=%d, Address=%s, Key=%s\n",
			currentNodeID, currentNodeInfo.Address, currentNodeInfo.Key)
	}

	// Ensure current node is in the room's peer list
	currentPeers := peerDetails.GetPeersInRoom(roomID)

	// Check if current node is already in the room
	currentNodeInRoom := false
	for _, peer := range currentPeers {
		if peer.PublicKey == currentNodeInfo.Key {
			currentNodeInRoom = true
			break
		}
	}

	// Add current node to room if not present
	if !currentNodeInRoom {
		err := peerDetails.AddPeer(currentNodeInfo.Key, currentNodeInfo.Address, true, roomID)
		if err != nil {
			fmt.Printf("Warning: Failed to add current node to room: %v\n", err)
		} else {
			fmt.Printf("Added current node to room %s\n", roomID)
		}
	}

	peers := peerDetails.GetPeersInRoom(roomID)

	// Translating public keys to uint34 IDs
	for _, peer := range peers {
		peerIDs = append(peerIDs, PublicKeyToID(peer.PublicKey))
	}

	// Sort nodeIDs in ascending order
	sort.Slice(peerIDs, func(i, j int) bool {
		return peerIDs[i] < peerIDs[j]
	})

	for i, id := range peerIDs {
		fmt.Printf("Node original ID %d assigned index ID %d\n", id, i)
		nodeIDs = append(nodeIDs, uint64(i))
	}

	if blockChainState {
		nodes = consensus.InitializeConsensus(len(nodeIDs), nodeIDs)
	}

	// Start the integrated network
	gossipNet.Start()

	return gossipNet, nil // Return the instance
}

// Start initializes and starts the gossip network
func (gn *GossipNetwork) Start() {
	// Start gossip protocol
	// go gn.gossipLoop()
	// go gn.healthCheckLoop()

	// Start network listener
	go gn.startListener()

	fmt.Println("Integrated gossip network started")
}

// startListener starts listening for incoming connections
func (gn *GossipNetwork) startListener() {
	var yggdrasilNodeInfo = GetYggdrasilNodeInfo()

	address := fmt.Sprintf("[%s]:%d", yggdrasilNodeInfo.Address, gn.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", gn.port, err)
	}
	gn.listener = listener
	defer listener.Close()

	log.Printf("Listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		go gn.handleConnection(conn)
	}
}

// DisableAuthentication turns off message authentication (for testing)
func (gn *GossipNetwork) DisableAuthentication() {
	gn.disableAuth = true
	fmt.Println("Authentication disabled for gossip network")
}

// EnableAuthentication turns on message authentication
func (gn *GossipNetwork) EnableAuthentication() {
	gn.disableAuth = false
	fmt.Println("Authentication enabled for gossip network")
}

// handleConnection handles incoming network connections
func (gn *GossipNetwork) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Connection received from %s", remoteAddr)

	// Read data from the connection
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from %s: %v", remoteAddr, err)
		return
	}

	// First try to parse as gossip message
	var gossipMsg GossipMessage
	if err := json.Unmarshal(buffer[:n], &gossipMsg); err == nil && gossipMsg.Type != "" {
		// This is a gossip message
		fmt.Printf("Received gossip message: %+v\n", gossipMsg)

		// Handle the message (authentication happens inside HandleGossipMessage)
		gn.HandleGossipMessage(gossipMsg)
		return
	}

	// Handle other message types as needed
	fmt.Printf("Received non-gossip message: %s\n", string(buffer[:n]))
}

// SendGossipMessage sends a gossip message to a specific peer
func (gn *GossipNetwork) SendGossipMessage(peer *GossipNode, msg GossipMessage) error {
	// Convert message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal gossip message: %v", err)
	}

	// Format address properly for IPv6
	var address string
	if strings.Contains(peer.Address, ":") && !strings.HasPrefix(peer.Address, "[") {
		// IPv6 address - wrap in square brackets
		address = fmt.Sprintf("[%s]:%d", peer.Address, gn.port)
	} else {
		// IPv4 address or already formatted
		address = fmt.Sprintf("%s:%d", peer.Address, gn.port)
	}

	// Connect to peer
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", address, err)
	}
	defer conn.Close()

	// Send the message
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send message to peer %s: %v", address, err)
	}

	fmt.Printf("Sent gossip message to peer %d at %s\n", peer.ID, address)
	return nil
}

// Gossip Protocol Functions

// gossipLoop is the main gossip loop
// func (gn *GossipNetwork) gossipLoop() {
// 	ticker := time.NewTicker(5 * time.Second)
// 	for range ticker.C {
// 		gn.gossipRandom()
// 	}
// }

// gossipRandom randomly gossips to subset of peers
// func (gn *GossipNetwork) gossipRandom() {
// 	gn.gossipMutex.RLock()
// 	peers := make([]*GossipNode, 0, len(gn.gossipPeers))
// 	for _, peer := range gn.gossipPeers {
// 		if peer.IsAlive {
// 			peers = append(peers, peer)
// 		}
// 	}

// 	gn.gossipMutex.RUnlock()

// 	if len(peers) == 0 {
// 		return
// 	}

// 	// Select random subset (typically 2-3 peers)
// 	numToGossip := min(2, len(peers))
// 	selected := gn.selectRandomPeers(peers, numToGossip)

// 	// Gossip heartbeat messages
// 	for _, peer := range selected {
// 		gn.sendHeartbeat(peer)
// 	}
// }

// selectRandomPeers selects a random subset of peers
func (gn *GossipNetwork) selectRandomPeers(peers []*GossipNode, count int) []*GossipNode {
	if count >= len(peers) {
		return peers
	}

	selected := make([]*GossipNode, count)
	indices := make(map[int]bool)

	for i := 0; i < count; i++ {
		for {
			idx := rand.Intn(len(peers))
			if !indices[idx] {
				indices[idx] = true
				selected[i] = peers[idx]
				break
			}
		}
	}

	return selected
}

// sendHeartbeat sends a heartbeat message to all peers
func (gn *GossipNetwork) sendHeartbeat() {
	heartbeatMsg := GossipMessage{
		ID:        generateMessageID(),
		Type:      "heartbeat",
		Data:      "ping",
		Sender:    fmt.Sprintf("%d", gn.nodeID),
		RoomID:    gn.roomID,
		Timestamp: time.Now().Unix(),
		TTL:       1,
		PublicKey: hex.EncodeToString(gn.publicKey),
	}

	// Sign the message
	messageBytes := []byte(fmt.Sprintf("%s%s%s%d", heartbeatMsg.ID, heartbeatMsg.Type, heartbeatMsg.Data, heartbeatMsg.Timestamp))

	signature := gn.SignMessage(messageBytes)
	heartbeatMsg.Signature = hex.EncodeToString(signature)

	// Send to all peers
	for _, peer := range gn.gossipPeers {
		if peer.ID != gn.nodeID {
			if err := gn.SendGossipMessage(peer, heartbeatMsg); err != nil {
				fmt.Printf("Failed to send heartbeat to peer %d: %v\n", peer.ID, err)
			}
		}
	}
}

// HandleGossipMessage handles incoming gossip messages
func (gn *GossipNetwork) HandleGossipMessage(msg GossipMessage) {
	// Check if we've seen this message
	gn.gossipMutex.Lock()
	if gn.messageHistory[msg.ID] {
		gn.gossipMutex.Unlock()
		return // Already seen
	}
	gn.messageHistory[msg.ID] = true
	gn.gossipMutex.Unlock()

	// Skip own heartbeat messages
	if msg.Type == "heartbeat" && msg.PublicKey == hex.EncodeToString(gn.publicKey) {
		return
	}

	// Process acknowledgments without signature verification
	if msg.Type == "ack" {
		if gn.isInRoom(msg.PublicKey) {
			gn.processGossipMessage(msg)
		}
		// Don't return here - continue to forward the ack to other nodes
	}

	// Find peer for authentication
	var peer GossipNode
	found := false
	for _, p := range gn.gossipPeers {
		if p.PublicKey == msg.PublicKey {
			peer = *p
			found = true
			break
		}
	}

	if !found {
		return
	}

	// Authenticate the message
	if !gn.authenticateMessage(msg, peer) {
		log.Printf("Message authentication failed for message ID: %s", msg.ID)
		return
	}

	// Determine if we should process this message
	shouldProcess := gn.shouldProcessMessage(msg)

	if shouldProcess {
		// Process the message for ourselves
		gn.processGossipMessage(msg)
	}

	// Always forward if TTL > 0
	if msg.TTL > 0 {
		msg.TTL--
		gn.forwardGossipMessage(msg)
	}
}

func (gn *GossipNetwork) shouldProcessMessage(msg GossipMessage) bool {
	// For now, process all messages in the room
	return msg.RoomID == gn.roomID
}

func (gn *GossipNetwork) isInRoom(publicKey string) bool {
	// Check if the public key is in the room's peer list
	peers := peerDetails.GetPeersInRoom(gn.roomID)
	for _, peer := range peers {
		if peer.PublicKey == publicKey {
			return true
		}
	}
	return false
}

func (gn *GossipNetwork) processGossipMessage(msg GossipMessage) {
	fmt.Printf("Processing gossip message from %s: %+v\n", msg.Sender, msg.Data)

	// Handle different message types
	switch msg.Type {
	case "chat":
		fmt.Printf("Processing chat message: %v\n", msg.Data)
		fmt.Println(msg.Sender)

		// Reachability check
		if msg.Data == "I hope I don't get censored!" {
			network.SendMessageToStatCollector("Message Reached To Peer "+hex.EncodeToString(gn.publicKey), msg.RoomID, 3001)
		}

		// Send acknowledgment
		senderID, _ := strconv.ParseUint(msg.Sender, 10, 64)
		ackData := fmt.Sprintf("ACK for message %s from %s", msg.ID, msg.PublicKey)
		gn.GossipMessage("ack", "broadcast", ackData, senderID, msg.RoomID, "")

		if blockChainState {
			// Check if we already have this message in processing
			messageAlreadyProcessing := false
			for _, processingMsg := range gn.msgsToProcess {
				if processingMsg.ID == msg.ID {
					messageAlreadyProcessing = true
					break
				}
			}

			// Only add to processing if we don't already have it
			if !messageAlreadyProcessing {
				// Initialize acks received counter
				msg.AcksReceived = 0
				gn.msgsToProcess = append(gn.msgsToProcess, msg)

				// Start goroutine to wait for acks
				go func(messageID string) {
					// Check every second instead of waiting full 15 seconds
					for i := 0; i < 15; i++ {
						time.Sleep(1 * time.Second)

						// Find the message and check if it received enough acks
						for j, processingMsg := range gn.msgsToProcess {
							if processingMsg.ID == messageID {
								if processingMsg.AcksReceived >= thresholdAcks {
									// Save to blockchain
									chatBlockData := fmt.Sprintf("CHAT_MSG{Sender: %s, Type: %s, Data: %s, Timestamp: %d, Signature: %s}",
										processingMsg.PublicKey, processingMsg.Type, processingMsg.Data, processingMsg.Timestamp, processingMsg.Signature)

									if err := blockchain.AddBlock(chatBlockData, msg.RoomID); err != nil {
										fmt.Printf("Failed to save chat message to blockchain: %v\n", err)
									} else {
										fmt.Printf("Yay\n")
									}

									// Remove from processing list
									gn.msgsToProcess = append(gn.msgsToProcess[:j], gn.msgsToProcess[j+1:]...)
									return // Exit early once threshold is reached
								}
								break
							}
						}
					}

					// If we reach here, the message didn't get enough acks in time
					for i, processingMsg := range gn.msgsToProcess {
						if processingMsg.ID == messageID {
							fmt.Printf("Message %s did not receive enough acks within timeout period. Received: %d, Required: %d\n", messageID, processingMsg.AcksReceived, thresholdAcks)
							// Remove from processing list
							gn.msgsToProcess = append(gn.msgsToProcess[:i], gn.msgsToProcess[i+1:]...)
							break
						}
					}
				}(msg.ID)
			}
		}

	case "ack":
		// Extract message ID from acknowledgment
		parts := strings.Split(msg.Data, " ")
		var messageID string
		if len(parts) >= 4 && strings.Contains(msg.Data, "ACK for message ") {
			messageID = parts[3] // "ACK for message <id> from <publicKey>"
		}

		fmt.Printf("Processing ack for message ID: %s\n", messageID)
		fmt.Printf("Current msgsToProcess count: %d\n", len(gn.msgsToProcess))

		if blockChainState {
			// Check if we've already processed this ack from this peer
			if gn.processedAcks[messageID] == nil {
				gn.processedAcks[messageID] = make(map[string]bool)
			}

			if !gn.processedAcks[messageID][msg.PublicKey] {
				// Mark this ack as processed from this peer
				gn.processedAcks[messageID][msg.PublicKey] = true

				// Find the message by the extracted message ID and increment counter
				if messageID != "" {
					found := false
					for i, processingMsg := range gn.msgsToProcess {
						fmt.Printf("Checking message %s against %s\n", processingMsg.ID, messageID)
						if processingMsg.ID == messageID {
							gn.msgsToProcess[i].AcksReceived++
							fmt.Printf("Message %s has received %d acks, threshold is %d\n", messageID, gn.msgsToProcess[i].AcksReceived, thresholdAcks)

							// Check if threshold is reached and save immediately
							if gn.msgsToProcess[i].AcksReceived >= thresholdAcks {
								// Save to blockchain
								chatBlockData := fmt.Sprintf("CHAT_MSG{Sender: %s, Type: %s, Data: %s, Timestamp: %d, Signature: %s}",
									processingMsg.PublicKey, processingMsg.Type, processingMsg.Data, processingMsg.Timestamp, processingMsg.Signature)

								if err := blockchain.AddBlock(chatBlockData, msg.RoomID); err != nil {
									fmt.Printf("Failed to save chat message to blockchain: %v\n", err)
								} else {
									fmt.Printf("Yay\n")
								}

								// Remove from processing list
								gn.msgsToProcess = append(gn.msgsToProcess[:i], gn.msgsToProcess[i+1:]...)
							}
							found = true
							break
						}
					}

					if !found {
						fmt.Printf("WARNING: Message %s not found in msgsToProcess list!\n", messageID)
					}
				}

				// Only save acknowledgment to blockchain if it's the first time from this peer
				ackBlockData := fmt.Sprintf("ACK_MSG{Sender: %s, Type: %s, Data: %s, Timestamp: %d, Signature: %s}",
					msg.PublicKey, msg.Type, msg.Data, msg.Timestamp, msg.Signature)

				if err := blockchain.AddBlock(ackBlockData, msg.RoomID); err != nil {
					fmt.Printf("Failed to save ack message to blockchain: %v\n", err)
				}
			}
		}

		fmt.Println("Received ack from: ")
		fmt.Println(msg.Sender)

		// case "heartbeat":
		// 	// Update peer last seen time
		// 	gn.updatePeerLastSeen(msg.OriginID)

		// case "consensus_message":
		// 	// Handle consensus-related gossip
		// 	gn.handleConsensusGossip(msg)
		// }
	}
}

// healthCheckLoop monitors peer health
// func (gn *GossipNetwork) healthCheckLoop() {
// 	ticker := time.NewTicker(30 * time.Second) // Changed from 15 to 30 seconds
// 	for range ticker.C {
// 		gn.checkPeerHealth()
// 	}
// }

// checkPeerHealth checks if peers are still alive
// func (gn *GossipNetwork) checkPeerHealth() {
// 	gn.gossipMutex.Lock()
// 	defer gn.gossipMutex.Unlock()

// 	now := time.Now()
// 	deadCount := 0
// 	aliveCount := 0

// 	for id, peer := range gn.gossipPeers {
// 		// Increase timeout from 60 seconds to 5 minutes
// 		if now.Sub(peer.LastSeen) > 5*60*time.Second {
// 			peer.IsAlive = false
// 			deadCount++
// 			fmt.Printf("Peer %d marked as dead (last seen: %v ago)\n", id, now.Sub(peer.LastSeen))
// 		} else {
// 			aliveCount++
// 		}
// 	}

// 	fmt.Printf("Peer health check: %d alive, %d dead, total: %d\n", aliveCount, deadCount, len(gn.gossipPeers))
// }

// updatePeerLastSeen updates peer last seen time
// func (gn *GossipNetwork) updatePeerLastSeen(peerID uint64) {
// 	gn.gossipMutex.Lock()
// 	defer gn.gossipMutex.Unlock()

// 	if peer, exists := gn.gossipPeers[peerID]; exists {
// 		peer.LastSeen = time.Now()
// 		peer.IsAlive = true
// 		fmt.Printf("Updated peer %d last seen time\n", peerID)
// 	}
// }

// Add a method to manually refresh peer health
// func (gn *GossipNetwork) RefreshPeerHealth() {
// 	gn.gossipMutex.Lock()
// 	defer gn.gossipMutex.Unlock()

// 	now := time.Now()
// 	for _, peer := range gn.gossipPeers {
// 		// Mark all peers as alive and update their last seen time
// 		peer.IsAlive = true
// 		peer.LastSeen = now
// 	}
// 	fmt.Printf("Refreshed health for all %d peers\n", len(gn.gossipPeers))
// }

// handleConsensusGossip handles consensus messages received via gossip
func (gn *GossipNetwork) handleConsensusGossip(msg GossipMessage) {
	// Handle consensus messages received via gossip
	fmt.Printf("Handling consensus gossip message: %+v\n", msg)
}

// forwardGossipMessage forwards gossip messages to other peers
func (gn *GossipNetwork) forwardGossipMessage(msg GossipMessage) {

	// simulating a targeted censorship
	if !isCensorshipAttackerNode {
		gn.gossipMutex.RLock()
		peers := make([]*GossipNode, 0, len(gn.gossipPeers))
		for _, peer := range gn.gossipPeers {
			if peer.IsAlive && peer.ID != gn.nodeID { // Fix: compare with nodeID instead of msg.Sender
				peers = append(peers, peer)
			}
		}
		gn.gossipMutex.RUnlock()

		// Forward to random subset
		numToForward := min(2, len(peers))
		selected := gn.selectRandomPeers(peers, numToForward)

		for _, peer := range selected {
			fmt.Printf("Forwarding gossip message to peer %d\n", peer.ID)
			gn.SendGossipMessage(peer, msg)
		}
	}

}

// GossipMessage sends a custom message to the network
func (gn *GossipNetwork) GossipMessage(msgType, category string, data interface{}, targetID uint64, roomID string, publicKeyStr string) {
	// Get our public key as hex
	publicKeyHex := hex.EncodeToString(gn.publicKey)

	// Create the message first
	gossipMsg := GossipMessage{
		ID:        generateMessageID(),
		PublicKey: publicKeyHex,
		Type:      msgType,
		Category:  category,
		Data:      fmt.Sprintf("%v", data), // Convert interface{} to string
		Sender:    fmt.Sprintf("%d", gn.nodeID),
		TargetID:  fmt.Sprintf("%d", targetID), // Convert uint64 to string
		RoomID:    roomID,
		Timestamp: time.Now().Unix(), // Use Unix timestamp instead of string
		TTL:       10,                // Set default TTL
	}

	// Sign the message
	messageBytes := []byte(fmt.Sprintf("%s%s%s%d", gossipMsg.ID, gossipMsg.Type, gossipMsg.Data, gossipMsg.Timestamp))

	signature := gn.SignMessage(messageBytes)
	gossipMsg.Signature = hex.EncodeToString(signature)

	// Add to our own processing first
	gn.HandleGossipMessage(gossipMsg)
}

// Yggdrasil Integration Functions

// GetYggdrasilPeers returns a list of Yggdrasil peer IP addresses
func (gn *GossipNetwork) GetYggdrasilPeers(mode string) ([]Peer, error) {
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getPeers").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run yggdrasilctl: %v", err)
	}

	var peerList PeerList
	if err := json.Unmarshal(out, &peerList); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	switch mode {
	case "inbound":
		inPeers := []Peer{}
		for _, p := range peerList.Peers {
			if p.Inbound {
				inPeers = append(inPeers, p)
			}
		}
		return inPeers, nil

	case "outbound":
		outPeers := []Peer{}
		for _, p := range peerList.Peers {
			if !p.Inbound {
				outPeers = append(outPeers, p)
			}
		}
		return outPeers, nil

	case "unique":
		uniquePeers := []Peer{}
		for _, p := range peerList.Peers {
			if !isPeerInList(p, uniquePeers) {
				uniquePeers = append(uniquePeers, p)
			}
		}
		return uniquePeers, nil
	}

	return nil, nil
}

// GetYggdrasilNodeInfo gets information about the current Yggdrasil node
func GetYggdrasilNodeInfo() YggdrasilNodeInfo {
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getSelf").Output()
	if err != nil {
		log.Fatalf("Failed to run yggdrasilctl: %v", err)
	}

	var yggdrasilNodeInfo YggdrasilNodeInfo
	if err := json.Unmarshal(out, &yggdrasilNodeInfo); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	return yggdrasilNodeInfo
}

// PublicKeyToID converts a hex-encoded ed25519 public key string into a deterministic uint64 ID
func (gn *GossipNetwork) PublicKeyToID(hexStr string) uint64 {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("invalid hex string: %v", err)
	}

	if len(bytes) != ed25519.PublicKeySize {
		log.Fatalf("invalid key length: expected %d, got %d", ed25519.PublicKeySize, len(bytes))
	}

	hash := sha256.Sum256(bytes)
	return binary.LittleEndian.Uint64(hash[:8])
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateMessageID() string {
	b := make([]byte, 16)
	cryptorand.Read(b)
	return hex.EncodeToString(b)
}

func isPeerInList(item Peer, list []Peer) bool {
	for _, v := range list {
		if v.Key == item.Key {
			return true // Item found, it's in the list
		}
	}
	return false // Item not found, it's not in the list
}

// PublicKeyToID converts a hex-encoded ed25519 public key string into a deterministic uint64 ID.
func PublicKeyToID(hexStr string) uint64 {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("invalid hex string: %v", err)
	}

	if len(bytes) != ed25519.PublicKeySize {
		log.Fatalf("invalid key length: expected %d, got %d", ed25519.PublicKeySize, len(bytes))
	}

	hash := sha256.Sum256(bytes)
	return binary.LittleEndian.Uint64(hash[:8])
}

func PublicKeyToNodeID(hexStr string) uint64 {

	ID := PublicKeyToID(hexStr)

	for i, peerID := range peerIDs {
		if ID == peerID {
			return uint64(i)
		}
	}

	return uint64(0)
}

// Function to find the index of a message by DigitalSignature
func FindMessageIndex(lookupSignature string) int {
	for i, msg := range msgsToProcess {
		if msg.Signature == lookupSignature {
			return i
		}
	}
	// Return -1 if no matching message is found
	return -1
}
