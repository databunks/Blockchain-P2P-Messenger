package gossipnetwork

// package network

// import (
// 	"crypto/rand"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"math/rand"
// 	"sync"
// 	"time"
// )

// // Gossip Protocol Types
// type GossipMessage struct {
// 	Type      string      `json:"type"`
// 	Category  string      `json:"category"` // "broadcast", "direct", "room_specific"
// 	Data      interface{} `json:"data"`
// 	OriginID  uint64      `json:"origin_id"`
// 	TargetID  uint64      `json:"target_id"`
// 	RoomID    string      `json:"room_id"`
// 	MessageID string      `json:"message_id"`
// 	TTL       int         `json:"ttl"`
// 	Timestamp int64       `json:"timestamp"`
// }

// type GossipNode struct {
// 	ID       uint64
// 	Address  string
// 	LastSeen time.Time
// 	IsAlive  bool
// }

// // GossipProtocol adds gossip capabilities to the existing Network
// type GossipProtocol struct {
// 	gossipPeers    map[uint64]*GossipNode
// 	messageHistory map[string]bool // Track seen messages
// 	gossipMutex    sync.RWMutex
// 	nodeID         uint64
// }

// // NewGossipProtocol creates a new gossip protocol instance
// func NewGossipProtocol(nodeID uint64) *GossipProtocol {
// 	return &GossipProtocol{
// 		nodeID:         nodeID,
// 		gossipPeers:    make(map[uint64]*GossipNode),
// 		messageHistory: make(map[string]bool),
// 	}
// }

// // AddPeer adds a peer to the gossip network
// func (gp *GossipProtocol) AddPeer(peerID uint64, address string) {
// 	gp.gossipMutex.Lock()
// 	defer gp.gossipMutex.Unlock()

// 	gp.gossipPeers[peerID] = &GossipNode{
// 		ID:       peerID,
// 		Address:  address,
// 		LastSeen: time.Now(),
// 		IsAlive:  true,
// 	}
// }

// // Start gossip protocol
// func (gp *GossipProtocol) Start() {
// 	go gp.gossipLoop()
// 	go gp.healthCheckLoop()
// 	fmt.Println("Gossip protocol started")
// }

// // Main gossip loop
// func (gp *GossipProtocol) gossipLoop() {
// 	ticker := time.NewTicker(2 * time.Second)
// 	for range ticker.C {
// 		gp.gossipRandom()
// 	}
// }

// // Randomly gossip to subset of peers
// func (gp *GossipProtocol) gossipRandom() {
// 	gp.gossipMutex.RLock()
// 	peers := make([]*GossipNode, 0, len(gp.gossipPeers))
// 	for _, peer := range gp.gossipPeers {
// 		if peer.IsAlive {
// 			peers = append(peers, peer)
// 		}
// 	}
// 	gp.gossipMutex.RUnlock()

// 	if len(peers) == 0 {
// 		return
// 	}

// 	// Select random subset (typically 3-5 peers)
// 	numToGossip := min(3, len(peers))
// 	selected := gp.selectRandomPeers(peers, numToGossip)

// 	// Gossip any pending messages
// 	for _, peer := range selected {
// 		gp.sendGossipMessage(peer)
// 	}
// }

// // Select random subset of peers
// func (gp *GossipProtocol) selectRandomPeers(peers []*GossipNode, count int) []*GossipNode {
// 	if count >= len(peers) {
// 		return peers
// 	}

// 	selected := make([]*GossipNode, count)
// 	indices := make(map[int]bool)

// 	for i := 0; i < count; i++ {
// 		for {
// 			idx := rand.Intn(len(peers))
// 			if !indices[idx] {
// 				indices[idx] = true
// 				selected[i] = peers[idx]
// 				break
// 			}
// 		}
// 	}

// 	return selected
// }

// // Send gossip message to a peer
// func (gp *GossipProtocol) sendGossipMessage(peer *GossipNode) {
// 	// Create gossip message
// 	msg := GossipMessage{
// 		Type:      "heartbeat",
// 		Category:  "broadcast",
// 		Data:      "Hello from gossip protocol",
// 		OriginID:  gp.nodeID,
// 		TargetID:  0,
// 		MessageID: generateMessageID(),
// 		TTL:       10,
// 		Timestamp: time.Now().Unix(),
// 	}

// 	// Send via existing network layer
// 	data, _ := json.Marshal(msg)
// 	fmt.Printf("Gossiping to peer %d: %s\n", peer.ID, string(data))
// }

// // Handle incoming gossip messages
// func (gp *GossipProtocol) HandleMessage(msg GossipMessage) {
// 	// Check if we've seen this message
// 	gp.gossipMutex.Lock()
// 	if gp.messageHistory[msg.MessageID] {
// 		gp.gossipMutex.Unlock()
// 		return // Already seen
// 	}
// 	gp.messageHistory[msg.MessageID] = true
// 	gp.gossipMutex.Unlock()

// 	// Determine if we should process this message
// 	shouldProcess := gp.shouldProcessMessage(msg)

// 	if shouldProcess {
// 		// Process the message for ourselves
// 		gp.processGossipMessage(msg)
// 	}

// 	// Always forward if TTL > 0 (unless it was meant only for us)
// 	if msg.TTL > 0 && msg.Category != "direct" {
// 		msg.TTL--
// 		gp.forwardGossipMessage(msg)
// 	}
// }

// func (gp *GossipProtocol) shouldProcessMessage(msg GossipMessage) bool {
// 	switch msg.Category {
// 	case "broadcast":
// 		return true // Everyone processes broadcasts

// 	case "direct":
// 		return msg.TargetID == gp.nodeID // Only target processes direct messages

// 	case "room_specific":
// 		// Check if we're in the specified room
// 		return gp.isInRoom(msg.RoomID)

// 	default:
// 		return false
// 	}
// }

// func (gp *GossipProtocol) isInRoom(roomID string) bool {
// 	// Implement room membership check
// 	// For now, return true
// 	return true
// }

// func (gp *GossipProtocol) processGossipMessage(msg GossipMessage) {
// 	fmt.Printf("Processing gossip message from %d: %+v\n", msg.OriginID, msg.Data)

// 	// Handle different message types
// 	switch msg.Type {
// 	case "chat":
// 		fmt.Printf("Processing chat message: %v\n", msg.Data)

// 	case "heartbeat":
// 		// Update peer last seen time
// 		gp.updatePeerLastSeen(msg.OriginID)

// 	case "consensus_message":
// 		// Handle consensus-related gossip
// 		gp.handleConsensusGossip(msg)
// 	}
// }

// func (gp *GossipProtocol) updatePeerLastSeen(peerID uint64) {
// 	gp.gossipMutex.Lock()
// 	defer gp.gossipMutex.Unlock()

// 	if peer, exists := gp.gossipPeers[peerID]; exists {
// 		peer.LastSeen = time.Now()
// 		peer.IsAlive = true
// 	}
// }

// func (gp *GossipProtocol) handleConsensusGossip(msg GossipMessage) {
// 	// Handle consensus messages received via gossip
// 	fmt.Printf("Handling consensus gossip message: %+v\n", msg)
// }

// // Forward gossip message to other peers
// func (gp *GossipProtocol) forwardGossipMessage(msg GossipMessage) {
// 	gp.gossipMutex.RLock()
// 	peers := make([]*GossipNode, 0, len(gp.gossipPeers))
// 	for _, peer := range gp.gossipPeers {
// 		if peer.IsAlive && peer.ID != msg.OriginID {
// 			peers = append(peers, peer)
// 		}
// 	}
// 	gp.gossipMutex.RUnlock()

// 	// Forward to random subset
// 	numToForward := min(2, len(peers))
// 	selected := gp.selectRandomPeers(peers, numToForward)

// 	for _, peer := range selected {
// 		data, _ := json.Marshal(msg)
// 		fmt.Printf("Forwarding gossip message to peer %d: %s\n", peer.ID, string(data))
// 	}
// }

// // Health check loop
// func (gp *GossipProtocol) healthCheckLoop() {
// 	ticker := time.NewTicker(10 * time.Second)
// 	for range ticker.C {
// 		gp.checkPeerHealth()
// 	}
// }

// // Check if peers are still alive
// func (gp *GossipProtocol) checkPeerHealth() {
// 	gp.gossipMutex.Lock()
// 	defer gp.gossipMutex.Unlock()

// 	for id, peer := range gp.gossipPeers {
// 		if time.Since(peer.LastSeen) > 30*time.Second {
// 			peer.IsAlive = false
// 			fmt.Printf("Peer %d marked as dead\n", id)
// 		}
// 	}
// }

// // Gossip a message to the network
// func (gp *GossipProtocol) GossipMessage(msgType, category string, data interface{}, targetID uint64, roomID string) {
// 	gossipMsg := GossipMessage{
// 		Type:      msgType,
// 		Category:  category,
// 		Data:      data,
// 		OriginID:  gp.nodeID,
// 		TargetID:  targetID,
// 		RoomID:    roomID,
// 		MessageID: generateMessageID(),
// 		TTL:       10,
// 		Timestamp: time.Now().Unix(),
// 	}

// 	// Add to our own processing first
// 	gp.HandleMessage(gossipMsg)
// }

// // Helper functions
// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }

// func generateMessageID() string {
// 	b := make([]byte, 16)
// 	rand.Read(b)
// 	return hex.EncodeToString(b)
// }
