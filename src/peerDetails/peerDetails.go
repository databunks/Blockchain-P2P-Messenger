package peerDetails

import (
	"encoding/json"
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

var peers []Peer

// AddPeer adds a new peer to the list and saves it to a file
func AddPeer(publicKey string, ip string, isAdmin bool) error {
	// Check for duplicate public key
	for _, peer := range peers {
		if peer.PublicKey == publicKey {
			return nil // Skip adding if public key already exists
		}
	}

	peer := Peer{
		PublicKey: publicKey,
		IP:        ip,
		IsAdmin:   isAdmin,
	}
	peers = append(peers, peer)
	return savePeersToFile()
}

// savePeersToFile saves the current list of peers to a file
func savePeersToFile() error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(dataDir, "peers.json"),
		os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read existing content
	var existingPeers []Peer
	if err := json.NewDecoder(file).Decode(&existingPeers); err != nil && !strings.Contains(err.Error(), "EOF") {
		return err
	}

	// Check for duplicates in existing peers
	for _, existingPeer := range existingPeers {
		if existingPeer.PublicKey == peers[len(peers)-1].PublicKey {
			return nil // Skip adding if public key already exists in file
		}
	}

	// Combine existing and new peers
	allPeers := append(existingPeers, peers[len(peers)-1])

	// Clear the file
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	// Write all peers back with proper formatting
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(allPeers)
}

// LoadPeersFromFile loads the list of peers from a file
func LoadPeersFromFile() error {
	file, err := os.Open(filepath.Join(dataDir, "peers.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No peers file exists yet
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&peers)
}
