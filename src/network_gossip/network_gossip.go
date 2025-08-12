package gossipnetwork

import (
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Gossip Protocol Types
type GossipMessage struct {
	PublicKey        string      `json:"public_key"`
	Type             string      `json:"type"`
	Category         string      `json:"category"` // "broadcast", "direct", "room_specific"
	Data             interface{} `json:"data"`
	OriginID         uint64      `json:"origin_id"`
	TargetID         uint64      `json:"target_id"`
	RoomID           string      `json:"room_id"`
	MessageID        string      `json:"message_id"`
	TTL              int         `json:"ttl"`
	Timestamp        uint64      `json:"timestamp"`
	DigitalSignature string      `json:"digital_signature"`
}

type GossipNode struct {
	ID       uint64
	Address  string
	LastSeen time.Time
	IsAlive  bool
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
	}
}

// loadKeys loads the private and public keys from keydetails.env
func loadKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	// Load environment variables from .env file
	err := godotenv.Load("keydetails.env")
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

// VerifyMessageSignature checks if the message was signed correctly using the sender's public key
func (gn *GossipNetwork) VerifyMessageSignature(messageContent []byte, signatureHex string, publicKeyHex string) bool {
	// Decode the hex-encoded public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		log.Printf("Error decoding public key: %v", err)
		return false
	}

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		log.Printf("Error decoding signature: %v", err)
		return false
	}

	// Check signature validity using ed25519
	isValid := ed25519.Verify(ed25519.PublicKey(publicKeyBytes), messageContent, signatureBytes)

	if !isValid {
		log.Println("Signature verification failed")
	}

	return isValid
}

// authenticateMessage checks if the message sender is authorized for the room
func (gn *GossipNetwork) authenticateMessage(msg GossipMessage) bool {
	if gn.disableAuth {
		return true
	}

	// Check if public key is in the room
	peers := peerDetails.GetPeersInRoom(msg.RoomID)
	for _, peer := range peers {
		if msg.PublicKey == peer.PublicKey {
			// Validate the digital signature
			messageBytes := []byte(fmt.Sprintf("%v", msg.Data))
			if !gn.VerifyMessageSignature(messageBytes, msg.DigitalSignature, peer.PublicKey) {
				log.Printf("Invalid signature for message from %s\nMessage: %v\nSignature: %s",
					msg.PublicKey, msg.Data, msg.DigitalSignature)
				return false
			}

			log.Printf("Message authenticated from %s", msg.PublicKey)
			return true
		}
	}

	log.Printf("Public key %s not found in room %s allow list", msg.PublicKey, msg.RoomID)
	return false
}

// InitializeGossipNetwork sets up the complete gossip network
func InitializeGossipNetwork(roomID string, port uint64) (*GossipNetwork, error) {
	fmt.Println("Initializing integrated gossip network...")

	// Create gossip network instance
	gossipNet := NewGossipNetwork(101, roomID, port)

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
			ID:       peerID,
			Address:  yggPeer.Address, // Use Yggdrasil IP address
			LastSeen: time.Now(),
			IsAlive:  true,
		}

		fmt.Printf("Added Yggdrasil peer to gossip network: ID=%d, Address=%s, Key=%s\n",
			peerID, yggPeer.Address, yggPeer.Key)
	}

	// Start the integrated network
	gossipNet.Start()

	return gossipNet, nil // Return the instance
}

// Start initializes and starts the gossip network
func (gn *GossipNetwork) Start() {
	// Start gossip protocol
	go gn.gossipLoop()
	go gn.healthCheckLoop()

	// Start network listener
	go gn.startListener()

	fmt.Println("Integrated gossip network started")
}

// startListener starts listening for incoming connections
func (gn *GossipNetwork) startListener() {
	var yggdrasilNodeInfo = gn.GetYggdrasilNodeInfo()

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
func (gn *GossipNetwork) gossipLoop() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		gn.gossipRandom()
	}
}

// gossipRandom randomly gossips to subset of peers
func (gn *GossipNetwork) gossipRandom() {
	gn.gossipMutex.RLock()
	peers := make([]*GossipNode, 0, len(gn.gossipPeers))
	for _, peer := range gn.gossipPeers {
		if peer.IsAlive {
			peers = append(peers, peer)
		}
	}
	gn.gossipMutex.RUnlock()

	if len(peers) == 0 {
		return
	}

	// Select random subset (typically 2-3 peers)
	numToGossip := min(2, len(peers))
	selected := gn.selectRandomPeers(peers, numToGossip)

	// Gossip heartbeat messages
	for _, peer := range selected {
		gn.sendHeartbeat(peer)
	}
}

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

// sendHeartbeat sends a heartbeat message to a peer
func (gn *GossipNetwork) sendHeartbeat(peer *GossipNode) {
	// Create message data
	messageData := "Hello from gossip protocol"
	messageBytes := []byte(messageData)

	// Sign the message
	signature := gn.SignMessage(messageBytes)
	signatureHex := hex.EncodeToString(signature)

	// Get our public key as hex
	publicKeyHex := hex.EncodeToString(gn.publicKey)

	msg := GossipMessage{
		PublicKey:        publicKeyHex,
		Type:             "heartbeat",
		Category:         "broadcast",
		Data:             messageData,
		OriginID:         gn.nodeID,
		TargetID:         0,
		MessageID:        generateMessageID(),
		TTL:              10,
		Timestamp:        uint64(time.Now().Unix()),
		DigitalSignature: signatureHex,
	}

	err := gn.SendGossipMessage(peer, msg)
	if err != nil {
		fmt.Printf("Failed to send heartbeat to peer %d: %v\n", peer.ID, err)
	}
}

// HandleGossipMessage handles incoming gossip messages
func (gn *GossipNetwork) HandleGossipMessage(msg GossipMessage) {
	// Check if we've seen this message
	gn.gossipMutex.Lock()
	if gn.messageHistory[msg.MessageID] {
		gn.gossipMutex.Unlock()
		return // Already seen
	}
	gn.messageHistory[msg.MessageID] = true
	gn.gossipMutex.Unlock()

	// Authenticate the message
	if !gn.authenticateMessage(msg) {
		log.Printf("Message authentication failed for message ID: %s", msg.MessageID)
		return
	}

	// Determine if we should process this message
	shouldProcess := gn.shouldProcessMessage(msg)

	if shouldProcess {
		// Process the message for ourselves
		gn.processGossipMessage(msg)
	}

	// Always forward if TTL > 0 (unless it was meant only for us)
	if msg.TTL > 0 && msg.Category != "direct" {
		msg.TTL--
		gn.forwardGossipMessage(msg)
	}
}

func (gn *GossipNetwork) shouldProcessMessage(msg GossipMessage) bool {
	switch msg.Category {
	case "broadcast":
		return true // Everyone processes broadcasts

	case "direct":
		return msg.TargetID == gn.nodeID // Only target processes direct messages

	case "room_specific":
		// Check if we're in the specified room
		return gn.isInRoom(msg.RoomID)

	default:
		return false
	}
}

func (gn *GossipNetwork) isInRoom(roomID string) bool {
	// Check if we're in the specified room
	return roomID == gn.roomID
}

func (gn *GossipNetwork) processGossipMessage(msg GossipMessage) {
	fmt.Printf("Processing gossip message from %d: %+v\n", msg.OriginID, msg.Data)

	// Handle different message types
	switch msg.Type {
	case "chat":
		fmt.Printf("Processing chat message: %v\n", msg.Data)

	case "heartbeat":
		// Update peer last seen time
		gn.updatePeerLastSeen(msg.OriginID)

	case "consensus_message":
		// Handle consensus-related gossip
		gn.handleConsensusGossip(msg)
	}
}

// healthCheckLoop monitors peer health
func (gn *GossipNetwork) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second) // Changed from 15 to 30 seconds
	for range ticker.C {
		gn.checkPeerHealth()
	}
}

// checkPeerHealth checks if peers are still alive
func (gn *GossipNetwork) checkPeerHealth() {
	gn.gossipMutex.Lock()
	defer gn.gossipMutex.Unlock()

	now := time.Now()
	deadCount := 0
	aliveCount := 0

	for id, peer := range gn.gossipPeers {
		// Increase timeout from 60 seconds to 5 minutes
		if now.Sub(peer.LastSeen) > 5*60*time.Second {
			peer.IsAlive = false
			deadCount++
			fmt.Printf("Peer %d marked as dead (last seen: %v ago)\n", id, now.Sub(peer.LastSeen))
		} else {
			aliveCount++
		}
	}

	fmt.Printf("Peer health check: %d alive, %d dead, total: %d\n", aliveCount, deadCount, len(gn.gossipPeers))
}

// updatePeerLastSeen updates peer last seen time
func (gn *GossipNetwork) updatePeerLastSeen(peerID uint64) {
	gn.gossipMutex.Lock()
	defer gn.gossipMutex.Unlock()

	if peer, exists := gn.gossipPeers[peerID]; exists {
		peer.LastSeen = time.Now()
		peer.IsAlive = true
		fmt.Printf("Updated peer %d last seen time\n", peerID)
	}
}

// Add a method to manually refresh peer health
func (gn *GossipNetwork) RefreshPeerHealth() {
	gn.gossipMutex.Lock()
	defer gn.gossipMutex.Unlock()

	now := time.Now()
	for _, peer := range gn.gossipPeers {
		// Mark all peers as alive and update their last seen time
		peer.IsAlive = true
		peer.LastSeen = now
	}
	fmt.Printf("Refreshed health for all %d peers\n", len(gn.gossipPeers))
}

// handleConsensusGossip handles consensus messages received via gossip
func (gn *GossipNetwork) handleConsensusGossip(msg GossipMessage) {
	// Handle consensus messages received via gossip
	fmt.Printf("Handling consensus gossip message: %+v\n", msg)
}

// forwardGossipMessage forwards gossip messages to other peers
func (gn *GossipNetwork) forwardGossipMessage(msg GossipMessage) {
	gn.gossipMutex.RLock()
	peers := make([]*GossipNode, 0, len(gn.gossipPeers))
	for _, peer := range gn.gossipPeers {
		if peer.IsAlive && peer.ID != msg.OriginID {
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

// GossipMessage sends a custom message to the network
func (gn *GossipNetwork) GossipMessage(msgType, category string, data interface{}, targetID uint64, roomID string) {
	// Create message data
	messageData := fmt.Sprintf("%v", data)
	messageBytes := []byte(messageData)

	// Sign the message
	signature := gn.SignMessage(messageBytes)
	signatureHex := hex.EncodeToString(signature)

	// Get our public key as hex
	publicKeyHex := hex.EncodeToString(gn.publicKey)

	gossipMsg := GossipMessage{
		PublicKey:        publicKeyHex,
		Type:             msgType,
		Category:         category,
		Data:             data,
		OriginID:         gn.nodeID,
		TargetID:         targetID,
		RoomID:           roomID,
		MessageID:        generateMessageID(),
		TTL:              10,
		Timestamp:        uint64(time.Now().Unix()),
		DigitalSignature: signatureHex,
	}

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
func (gn *GossipNetwork) GetYggdrasilNodeInfo() YggdrasilNodeInfo {
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
