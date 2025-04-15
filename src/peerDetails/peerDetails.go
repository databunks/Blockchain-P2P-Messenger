package peerDetails

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const dataDir = "data"

type Peer struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
	IsAdmin   bool   `json:"is_admin"`
}

var peers map[string][]Peer // roomID -> []Peer

func init() {
	peers = make(map[string][]Peer)
}

// AddPeer adds a new peer to a specific room
func AddPeer(publicKey string, ip string, isAdmin bool, roomID string) error {
	fmt.Printf("Adding peer: %s, %s, %v to room %s\n", publicKey, ip, isAdmin, roomID)

	// Create room directory if it doesn't exist
	roomDir := filepath.Join(dataDir, roomID)
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return err
	}

	// Check for duplicate public key in the room
	if roomPeers, exists := peers[roomID]; exists {
		for _, peer := range roomPeers {
			if peer.PublicKey == publicKey {
				fmt.Println("Peer already exists in room")
				return nil // Skip adding if public key already exists in this room
			}
		}
	}

	peer := Peer{
		PublicKey: publicKey,
		IP:        ip,
		IsAdmin:   isAdmin,
	}

	// Add to memory
	peers[roomID] = append(peers[roomID], peer)
	fmt.Printf("Added peer to memory. Current peers: %+v\n", peers[roomID])

	// Save to room-specific file
	return savePeersToFile(roomID)
}

// GetPeersInRoom returns all peers in a specific room
func GetPeersInRoom(roomID string) []Peer {
	if roomPeers, exists := peers[roomID]; exists {
		return roomPeers
	}
	return []Peer{}
}

// RemovePeerFromRoom removes a peer from a specific room
func RemovePeerFromRoom(publicKey string, roomID string) error {
	if roomPeers, exists := peers[roomID]; exists {
		var newPeers []Peer
		for _, peer := range roomPeers {
			if peer.PublicKey != publicKey {
				newPeers = append(newPeers, peer)
			}
		}
		peers[roomID] = newPeers
		return savePeersToFile(roomID)
	}
	return nil
}

// savePeersToFile saves the peers of a specific room to its file
func savePeersToFile(roomID string) error {
	fmt.Printf("Saving peers for room %s\n", roomID)
	fmt.Printf("Current peers: %+v\n", peers[roomID])

	roomDir := filepath.Join(dataDir, roomID)
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(roomDir, "peers.json")
	fmt.Printf("Saving to file: %s\n", filePath)

	file, err := os.OpenFile(filePath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(peers[roomID]); err != nil {
		return err
	}

	fmt.Println("Successfully saved peers to file")
	return nil
}

// LoadPeersFromFile loads all peers from all room files
func LoadPeersFromFile() error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Read all room directories
	rooms, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		if !room.IsDir() {
			continue
		}

		roomID := room.Name()
		file, err := os.Open(filepath.Join(dataDir, roomID, "peers.json"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer file.Close()

		var roomPeers []Peer
		if err := json.NewDecoder(file).Decode(&roomPeers); err != nil {
			if !strings.Contains(err.Error(), "EOF") {
				return err
			}
		}
		peers[roomID] = roomPeers
	}

	return nil
}
