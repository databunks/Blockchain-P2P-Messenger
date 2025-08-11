package gossipnetwork

import (
	"blockchain-p2p-messenger/src/consensus"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	"math/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
	"github.com/anthdm/hbbft"
	"github.com/joho/godotenv"
)

type Message struct {
	PublicKey        string `json:"public_key"`
	Message          string `json:"message"`
	Type             string `json:"type"`
	RoomID           string `json:"room_id"`
	DigitalSignature string `json:"digital_signature"`
	Timestamp        uint64 `json:"timestamp"`
}

type message struct {
	from    uint64
	payload hbbft.MessageTuple
}

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

// Gossip Protocol Types
type GossipMessage struct {
	Type      string      `json:"type"`
	Category  string      `json:"category"` // "broadcast", "direct", "room_specific"
	Data      interface{} `json:"data"`
	OriginID  uint64      `json:"origin_id"`
	TargetID  uint64      `json:"target_id"`
	RoomID    string      `json:"room_id"`
	MessageID string      `json:"message_id"`
	TTL       int         `json:"ttl"`
	Timestamp int64       `json:"timestamp"`
}

type GossipNode struct {
	ID       uint64
	Address  string
	LastSeen time.Time
	IsAlive  bool
}

var hb *hbbft.HoneyBadger
var idToPubKey = make(map[uint64]string)

// Network struct with gossip capabilities
type Network struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	peers      map[string]string // peerID -> IP
	conns      map[string]net.Conn
	msgChan    chan Message
	mu         sync.RWMutex

	// Gossip protocol fields
	gossipPeers    map[uint64]*GossipNode
	messageHistory map[string]bool // Track seen messages
	gossipMutex    sync.RWMutex
	nodeID         uint64
}

// HBMessage struct for HBBFT messages
type HBMessage struct {
	From  uint64
	Epoch uint64
	Data  []byte // serialized hbbft.ACSMessage
}

func InitializeNetwork(roomID string) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	for i := 0; i < 10; i++ {
		peersOnline, _ := GetYggdrasilPeers("unique")
		peers = peerDetails.GetPeersInRoom(roomID)

		peerSet := make(map[string]struct{})
		for _, p := range peersOnline {
			peerSet[p.Key] = struct{}{}
		}

		lenPeers := 0
		for _, p := range peers {
			if _, ok := peerSet[p.PublicKey]; ok {
				fmt.Println(p.PublicKey, "is online")
				lenPeers++
			} else {
				fmt.Println(p.PublicKey, "is offline")
			}
		}
		if lenPeers < 3 {
			fmt.Printf("Retrying Connection.... (attempt %d)\n", i)
			time.Sleep(5 * time.Second)
		} else if lenPeers >= 3 {
			fmt.Printf("Initializing consensus with Yggdrasil peers: %v\n", peers)
			break
		} else if i == 9 {
			return fmt.Errorf("error: Not enough nodes have been connected (%d nodes connected)", i)
		}
	}

	// Link peers to consensus (public key is passed into node id)
	var peerIDs []uint64

	// Translating public keys to uint64 IDs
	for _, peer := range peers {
		peerIDs = append(peerIDs, PublicKeyToID(peer.PublicKey))
	}

	cfg := hbbft.Config{
		N:         4,
		ID:        101,
		Nodes:     peerIDs,
		BatchSize: 1,
	}

	fmt.Println(len(peerIDs))
	hb = hbbft.NewHoneyBadger(cfg)
	go hb.Start()

	// Initialize gossip protocol
	net := &Network{
		nodeID:         101,
		gossipPeers:    make(map[uint64]*GossipNode),
		messageHistory: make(map[string]bool),
	}

	// Add peers to gossip network
	for _, peer := range peers {
		peerID := PublicKeyToID(peer.PublicKey)
		net.gossipPeers[peerID] = &GossipNode{
			ID:       peerID,
			Address:  peer.IP,
			LastSeen: time.Now(),
			IsAlive:  true,
		}
	}

	// Start gossip protocol
	net.StartGossip()

	ListenOnPort(3000)

	return nil
}

// Gossip Protocol Functions

// Start gossip protocol
func (n *Network) StartGossip() {
	go n.gossipLoop()
	go n.healthCheckLoop()
	fmt.Println("Gossip protocol started")
}

// Main gossip loop
func (n *Network) gossipLoop() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		n.gossipRandom()
	}
}

// Randomly gossip to subset of peers
func (n *Network) gossipRandom() {
	n.gossipMutex.RLock()
	peers := make([]*GossipNode, 0, len(n.gossipPeers))
	for _, peer := range n.gossipPeers {
		if peer.IsAlive {
			peers = append(peers, peer)
		}
	}
	n.gossipMutex.RUnlock()

	if len(peers) == 0 {
		return
	}

	// Select random subset (typically 3-5 peers)
	numToGossip := min(3, len(peers))
	selected := n.selectRandomPeers(peers, numToGossip)

	// Gossip any pending messages
	for _, peer := range selected {
		n.sendGossipMessage(peer)
	}
}

// Select random subset of peers
func (n *Network) selectRandomPeers(peers []*GossipNode, count int) []*GossipNode {
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

// Send gossip message to a peer
func (n *Network) sendGossipMessage(peer *GossipNode) {
	// Create gossip message
	msg := GossipMessage{
		Type:      "heartbeat",
		Category:  "broadcast",
		Data:      "Hello from gossip protocol",
		OriginID:  n.nodeID,
		TargetID:  0,
		MessageID: generateMessageID(),
		TTL:       10,
		Timestamp: time.Now().Unix(),
	}

	// Send via existing network layer
	data, _ := json.Marshal(msg)
	// Note: You'll need to implement the actual sending mechanism
	fmt.Printf("Gossiping to peer %d: %s\n", peer.ID, string(data))
}

// Handle incoming gossip messages
func (n *Network) handleGossipMessage(msg GossipMessage) {
	// Check if we've seen this message
	n.gossipMutex.Lock()
	if n.messageHistory[msg.MessageID] {
		n.gossipMutex.Unlock()
		return // Already seen
	}
	n.messageHistory[msg.MessageID] = true
	n.gossipMutex.Unlock()

	// Determine if we should process this message
	shouldProcess := n.shouldProcessMessage(msg)

	if shouldProcess {
		// Process the message for ourselves
		n.processGossipMessage(msg)
	}

	// Always forward if TTL > 0 (unless it was meant only for us)
	if msg.TTL > 0 && msg.Category != "direct" {
		msg.TTL--
		n.forwardGossipMessage(msg)
	}
}

func (n *Network) shouldProcessMessage(msg GossipMessage) bool {
	switch msg.Category {
	case "broadcast":
		return true // Everyone processes broadcasts

	case "direct":
		return msg.TargetID == n.nodeID // Only target processes direct messages

	case "room_specific":
		// Check if we're in the specified room
		return n.isInRoom(msg.RoomID)

	default:
		return false
	}
}

func (n *Network) isInRoom(roomID string) bool {
	// Implement room membership check
	// For now, return true
	return true
}

func (n *Network) processGossipMessage(msg GossipMessage) {
	fmt.Printf("Processing gossip message from %d: %+v\n", msg.OriginID, msg.Data)

	// Handle different message types
	switch msg.Type {
	case "chat":
		// Add to consensus
		if chatData, ok := msg.Data.(string); ok {
			tx := consensus.NewTransaction("", "chat", "", uint64(time.Now().Unix()), "")
			hb.AddTransaction(tx)
			fmt.Printf("Added chat message to consensus: %s\n", chatData)
		}

	case "heartbeat":
		// Update peer last seen time
		n.updatePeerLastSeen(msg.OriginID)

	case "consensus_message":
		// Handle consensus-related gossip
		n.handleConsensusGossip(msg)
	}
}

func (n *Network) updatePeerLastSeen(peerID uint64) {
	n.gossipMutex.Lock()
	defer n.gossipMutex.Unlock()

	if peer, exists := n.gossipPeers[peerID]; exists {
		peer.LastSeen = time.Now()
		peer.IsAlive = true
	}
}

func (n *Network) handleConsensusGossip(msg GossipMessage) {
	// Handle consensus messages received via gossip
	fmt.Printf("Handling consensus gossip message: %+v\n", msg)
}

// Forward gossip message to other peers
func (n *Network) forwardGossipMessage(msg GossipMessage) {
	n.gossipMutex.RLock()
	peers := make([]*GossipNode, 0, len(n.gossipPeers))
	for _, peer := range n.gossipPeers {
		if peer.IsAlive && peer.ID != msg.OriginID {
			peers = append(peers, peer)
		}
	}
	n.gossipMutex.RUnlock()

	// Forward to random subset
	numToForward := min(2, len(peers))
	selected := n.selectRandomPeers(peers, numToForward)

	for _, peer := range selected {
		data, _ := json.Marshal(msg)
		fmt.Printf("Forwarding gossip message to peer %d: %s\n", peer.ID, string(data))
	}
}

// Health check loop
func (n *Network) healthCheckLoop() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		n.checkPeerHealth()
	}
}

// Check if peers are still alive
func (n *Network) checkPeerHealth() {
	n.gossipMutex.Lock()
	defer n.gossipMutex.Unlock()

	for id, peer := range n.gossipPeers {
		if time.Since(peer.LastSeen) > 30*time.Second {
			peer.IsAlive = false
			fmt.Printf("Peer %d marked as dead\n", id)
		}
	}
}

// Gossip a message to the network
func (n *Network) GossipMessage(msgType, category string, data interface{}, targetID uint64, roomID string) {
	gossipMsg := GossipMessage{
		Type:      msgType,
		Category:  category,
		Data:      data,
		OriginID:  n.nodeID,
		TargetID:  targetID,
		RoomID:    roomID,
		MessageID: generateMessageID(),
		TTL:       10,
		Timestamp: time.Now().Unix(),
	}

	// Add to our own processing first
	n.handleGossipMessage(gossipMsg)
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
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Function to check if an item is not in a list
func isNotInList(item Peer, list []Peer) bool {
	for _, v := range list {
		if v == item {
			return false // Item found, it's in the list
		}
	}
	return true // Item not found, it's not in the list
}

// GetYggdrasilPeers returns a list of Yggdrasil peer IP addresses
func GetYggdrasilPeers(mode string) ([]Peer, error) {
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getPeers").Output()

	if err != nil {
		log.Fatalf("Failed to run yggdrasilctl: %v", err)
	}

	var peerList PeerList
	if err := json.Unmarshal(out, &peerList); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	switch mode {
	case "inbound":
		inPeers := []Peer{}
		for _, p := range peerList.Peers {
			if p.Inbound {
				inPeers = append(inPeers, p)
			}
		}
		return inPeers, err

	case "outbound":
		outPeers := []Peer{}
		for _, p := range peerList.Peers {
			if !p.Inbound {
				outPeers = append(outPeers, p)
			}
		}
		return outPeers, err

	case "unique":
		uniquePeers := []Peer{}
		for _, p := range peerList.Peers {
			if isNotInList(p, uniquePeers) {
				uniquePeers = append(uniquePeers, p)
			}
		}
		return uniquePeers, err
	}

	return nil, nil
}

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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Connection received from %s", remoteAddr)

	// Read data from the connection
	buffer := make([]byte, 1024) // Buffer to store incoming data
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
		// You'll need to get the network instance to handle this
		return
	}

	// Try to parse as regular message
	var message Message
	err = json.Unmarshal(buffer[:n], &message)
	if err != nil {
		log.Printf("Error unmarshaling message: %v\n", err)
		return
	}
	message.Timestamp = uint64(time.Now().Unix())

	// First we check if public key is in the room
	peers := peerDetails.GetPeersInRoom(message.RoomID)
	for _, peer := range peers {
		log.Printf("Peers Public Key: %s\nMessage Public Key:%s\n", peer.PublicKey, message.PublicKey)
		if message.PublicKey == peer.PublicKey {
			// Validate the digital signature
			if !VerifyMessageSignature([]byte(message.Message), message.DigitalSignature, peer.PublicKey) {
				log.Printf("Invalid signature for message from %s\nMessage: %s\nSignature:%s", message.PublicKey, message.Message, message.DigitalSignature)

				responseMessage := "Invalid Digital Signature!!"
				conn.Write([]byte(responseMessage))
				return
			}

			log.Printf("Authenticated!")

			switch message.Type {
			case "chat":
				log.Printf(message.Message)

				// Add to consensus
				tx := consensus.NewTransaction(message.PublicKey, "chat", message.DigitalSignature, message.Timestamp, "")
				hb.AddTransaction(tx)

				// Get message outputs + send to whoever
				fmt.Println(hb.Messages())

				fmt.Println("Added Message as Transaction")
			}

			return
		}
	}

	responseMessage := "Public Key Not Found In Allow List!"
	conn.Write([]byte(responseMessage))
}

func ListenOnPort(port int) {
	var yggdrasilNodeInfo = GetYggdrasilNodeInfo()

	address := fmt.Sprintf("[%s]:%d", yggdrasilNodeInfo.Address, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		go handleConnection(conn)
	}
}

func SignMessage(message []byte) []byte {
	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the ED25519 private key from the environment variable
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("ED25519_PRIVATE_KEY is not set in .env file")
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

	// Sign a message using the private key
	signature := ed25519.Sign(privateKey, message)

	return signature
}

// VerifyMessageSignature checks if the message was signed correctly using the sender's public key
func VerifyMessageSignature(messageContent []byte, signatureHex string, publicKeyHex string) bool {
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

func SendMessage(messageContent string, roomID string, port uint64, typeofmessage string) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the ED25519 private key from the environment variable
	publicKeyHex := os.Getenv("PUBLIC_KEY")
	if publicKeyHex == "" {
		log.Fatal("ED25519_PUBLIC_KEY is not set in .env file")
	}

	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)
		go func(peer peerDetails.Peer) {
			defer wg.Done()

			// Create the message struct
			message := Message{
				PublicKey: publicKeyHex,
				Message:   messageContent,
				RoomID:    roomID,
				Type:      typeofmessage,
				Timestamp: uint64(time.Now().Unix()),
			}

			timestampBytes := []byte(fmt.Sprintf("%d", message.Timestamp))

			// Sign the message
			signature := SignMessage(append([]byte(messageContent), timestampBytes...))
			message.DigitalSignature = hex.EncodeToString(signature)

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
			msgBytes, err := json.Marshal(message)
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
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				log.Printf("Error reading response from %s: %v\n", peer.IP, err)
				return
			}

			// Print the response received from the peer
			response := string(buffer[:n])
			fmt.Printf("Response from %s: %s\n", peer.IP, response)

		}(peer)
	}

	wg.Wait()
	return nil
}

func StartYggdrasilServer() error {
	ListenOnPort(3000)
	return nil
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
