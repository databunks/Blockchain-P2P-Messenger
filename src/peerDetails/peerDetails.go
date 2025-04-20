package peerDetails

import (
	"blockchain-p2p-messenger/src/blockchain"
	"encoding/json"
	"fmt"
	"strings"
)

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

	peer := Peer{
		PublicKey: publicKey,
		IP:        ip,
		IsAdmin:   isAdmin,
	}

	// Get Previous Peers
	GetPeersInRoom(roomID)

	// Add to memory
	peers[roomID] = append(peers[roomID], peer)
	fmt.Printf("Added peer to memory. Current peers: %+v\n", peers[roomID])

	// Save to blockchain
	peerJson, err := json.Marshal(peers[roomID])
	if err != nil {
		return fmt.Errorf("failed to marshal peer list: %v", err)
	}

	blockchain.AddBlock(fmt.Sprintf("PEER_ADDED\n%s", peerJson), roomID)

		
	return nil
	// return savePeersToFile(roomID)
}

// GetPeersInRoom loads the latest peer list for a specific room from the blockchain
func GetPeersInRoom(roomID string) []Peer {
	current_blockchain := blockchain.GetBlockchain(roomID)

	for i := len(current_blockchain) - 1; i >= 0; i-- {
		block := current_blockchain[i]

		if strings.HasPrefix(block.Data, "PEER_") {
			var roomPeers []Peer
			data := strings.SplitN(block.Data, "\n", 2)
			if len(data) < 2 {
				break
			}

			if err := json.Unmarshal([]byte(data[1]), &roomPeers); err != nil {
				fmt.Println("Failed to unmarshal peer list:", err)
				break
			}

			// Update in-memory cache and return the peer list
			peers[roomID] = roomPeers
			return roomPeers
		}
	}

	// No peers found in blockchain for this room
	peers[roomID] = []Peer{}
	return peers[roomID]
}


// RemovePeer removes a peer by public key from a specific room
func RemovePeer(publicKey string, roomID string) error {
	fmt.Printf("Removing peer with public key: %s from room: %s\n", publicKey, roomID)

	// Load latest peers from blockchain
	currentPeers := GetPeersInRoom(roomID)

	// Filter out the peer we want to remove
	newPeerList := []Peer{}
	found := false
	for _, peer := range currentPeers {
		if peer.PublicKey != publicKey {
			newPeerList = append(newPeerList, peer)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("peer with public key %s not found in room %s", publicKey, roomID)
	}

	// Update memory
	peers[roomID] = newPeerList
	fmt.Printf("Updated peer list after removal: %+v\n", newPeerList)

	// Save new peer list to blockchain
	peerJson, err := json.Marshal(newPeerList)
	if err != nil {
		return fmt.Errorf("failed to marshal peer list: %v", err)
	}

	blockchain.AddBlock(fmt.Sprintf("PEER_REMOVED\n%s", peerJson), roomID)

	return nil
}



