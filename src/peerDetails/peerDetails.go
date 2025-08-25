package peerDetails

import (
	"blockchain-p2p-messenger/src/blockchain"
	"blockchain-p2p-messenger/src/derivationFunctions"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Peer struct {
	PublicKey string `json:"publickey"`
	IP        string `json:"ip"`
	IsAdmin   bool   `json:"isadmin"`
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
	peers[roomID] = GetPeersInRoom(roomID)

	// Small delay to prevent race condition
	time.Sleep(10 * time.Millisecond)

	// Add to memory
	peers[roomID] = append(peers[roomID], peer)
	fmt.Printf("Added peer to memory. Current peers: %+v\n", peers[roomID])

	peerList := "["
	for i := 0; i < len(peers[roomID]); i++ {
		peerList += "{"
		peerList += peers[roomID][i].PublicKey
		peerList += " " + peers[roomID][i].IP
		peerList += " " + fmt.Sprintf("%v", peers[roomID][i].IsAdmin)
		peerList += "}"
		if i != len(peers[roomID])-1 {
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

			// Return the peer list without updating global map (prevents concurrent writes)
			return roomPeers
		}
	}

	// No peers found in blockchain for this room
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
	// Small delay to prevent race condition
	time.Sleep(10 * time.Millisecond)
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
