package peerDetails

import (
	"blockchain-p2p-messenger/src/blockchain"
	"blockchain-p2p-messenger/src/derivationFunctions"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Peer struct {
	PublicKey string `json:"publickey"`
	IP        string `json:"ip"`
	IsAdmin   bool   `json:"isadmin"`
}

var peers sync.Map // roomID -> []Peer (thread-safe)

// AddPeer adds a new peer to a specific room
func AddPeer(publicKey string, ip string, isAdmin bool, roomID string) error {
	fmt.Printf("Adding peer: %s, %s, %v to room %s\n", publicKey, ip, isAdmin, roomID)

	peer := Peer{
		PublicKey: publicKey,
		IP:        ip,
		IsAdmin:   isAdmin,
	}

	// Get Previous Peers
	previousPeers := GetPeersInRoom(roomID)

	// Add to memory
	newPeers := append(previousPeers, peer)
	peers.Store(roomID, newPeers)
	fmt.Printf("Added peer to memory. Current peers: %+v\n", newPeers)

	peerList := "["
	for i := 0; i < len(newPeers); i++ {
		peerList += "{"
		peerList += newPeers[i].PublicKey
		peerList += " " + newPeers[i].IP
		peerList += " " + fmt.Sprintf("%v", newPeers[i].IsAdmin)
		peerList += "}"
		if i != len(newPeers)-1 {
			peerList += ", "
		}
	}

	peerList += "]"

	blockchain.AddBlock(fmt.Sprintf("PEER_ADDED$%s", peerList), roomID)

	return nil
}

// GetPeersInRoom loads the latest peer list for a specific room from the blockchain
func GetPeersInRoom(roomID string) []Peer {
	current_blockchain := blockchain.GetBlockchain(roomID)

	for i := len(current_blockchain) - 1; i >= 0; i-- {
		block := current_blockchain[i]

		if strings.HasPrefix(block.Data, "PEER_") {
			var roomPeers []Peer

			dataArray := strings.SplitN(block.Data, "$", 2)
			dataArray = strings.SplitN(dataArray[1], "{", -1)

			for i := 1; i < len(dataArray); i++ {

				var currentElement = strings.SplitN(dataArray[i], "}", 2)[0]
				peers := strings.SplitN(currentElement, " ", 3)

				isAdmin, err := strconv.ParseBool(peers[2])
				ipaddr := derivationFunctions.DeriveIPAddressFromPublicKey(peers[0])

				roomPeers = append(roomPeers, Peer{PublicKey: peers[0], IP: ipaddr, IsAdmin: isAdmin})

				if err != nil {
					fmt.Println("Error:", err)
				}
			}

			// Update in-memory cache and return the peer list
			peers.Store(roomID, roomPeers)
			return roomPeers
		}
	}

	// No peers found in blockchain for this room
	peers.Store(roomID, []Peer{})
	return []Peer{}
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
	peers.Store(roomID, newPeerList)
	fmt.Printf("Updated peer list after removal: %+v\n", newPeerList)

	// Save new peer list to blockchain
	peerJson, err := json.Marshal(newPeerList)
	if err != nil {
		return fmt.Errorf("failed to marshal peer list: %v", err)
	}

	blockchain.AddBlock(fmt.Sprintf("PEER_REMOVED\n%s", peerJson), roomID)

	return nil
}
