package room

import (
	"blockchain-p2p-messenger/src/auth"
	"blockchain-p2p-messenger/src/blockchain"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const dataDir = "data"

type Room struct {
	ID        string
	Name      string
	Members   []string // List of member public keys
	CreatedAt int64
}

// CreateRoom creates a new chat room and initializes it with the current peer and blockchain
func CreateRoom(name string) (*Room, error) {
	// Generate a random room ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}

	roomID := hex.EncodeToString(idBytes)
	room := &Room{
		ID:        roomID,
		Name:      name,
		Members:   make([]string, 0),
		CreatedAt: time.Now().Unix(),
	}

	// Create room directory
	roomDir := filepath.Join(dataDir, roomID)
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return nil, err
	}

	// Read keydetails.env
	keyDetails, err := os.ReadFile("../keydetails.env")
	if err != nil {
		return nil, err
	}

	fmt.Println("keyDetails", string(keyDetails))
	// Parse keydetails.env
	lines := strings.Split(string(keyDetails), "\n")
	var publicKey, ip string
	for _, line := range lines {
		if strings.HasPrefix(line, "PUBLIC_KEY=") {
			publicKey = strings.TrimPrefix(line, "PUBLIC_KEY=")
		} else if strings.HasPrefix(line, "IP=") {
			ip = strings.TrimPrefix(line, "IP=")
			fmt.Println("ip", ip)
		}
	}

	// Add current peer to the room
	if err := peerDetails.AddPeer(publicKey, ip, true, roomID); err != nil {
		return nil, err
	}

	// Initialize blockchain for the room
	if err := blockchain.LoadBlockchainFromFile(roomID); err != nil {
		return nil, err
	}

	chain := blockchain.GetBlockchain(roomID)
	auth.AddAdmin(publicKey, chain, roomID)

	return room, nil
}
