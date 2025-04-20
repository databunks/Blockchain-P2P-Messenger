package main

import "blockchain-p2p-messenger/src/network"

// "blockchain-p2p-messenger/src/consensus"
// "blockchain-p2p-messenger/src/peerDetails"
// "crypto/ed25519"
// "fmt"

func main() {
	// network.StartYggdrasilServer()
	// publicKey := "123abc456def7890"                // mock public key
	// publicKey2 := "joebiden"
	// publicKey3 := "joebiden2"
	// ip := "219:84b6:648e:9ca5:e124:49ed:42d2:e6a3" // fake IPv6 address
	// isAdmin := false
	// roomID := "room-xyz-987" // mock room ID

	// peerDetails.AddPeer(publicKey, ip, isAdmin, roomID)
	// peerDetails.AddPeer(publicKey2, ip, isAdmin, roomID)
	// peerDetails.AddPeer(publicKey3, ip, isAdmin, roomID)

	// peerDetails.RemovePeer(publicKey, roomID)
	//network.GetYggdrasilPeers()

	network.InitializeNetwork("room-xyz-987")
}
