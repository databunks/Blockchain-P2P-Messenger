package main

import (
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
	"log"
	"time"
)

//"blockchain-p2p-messenger/src/genkeys"
// "blockchain-p2p-messenger/src/simulations"
// "blockchain-p2p-messenger/src/consensus"

// "blockchain-p2p-messenger/src/simulations/consensus_tests"
// "crypto/ed25519"
// "fmt"

func main() {

	// consensustests.RunTest1()
	// network.StartYggdrasilServer()
	publicKey_VM1 := "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
	publicKey2_VM2 := "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
	publicKey3_VM3 := "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
	ip := "219:84b6:648e:9ca5:e124:49ed:42d2:e6a3" // fake IPv6 address
	isAdmin := false
	roomID := "room-xyz-987" // mock room IDd

	peerDetails.AddPeer(publicKey_VM1, ip, isAdmin, roomID)
	peerDetails.AddPeer(publicKey2_VM2, ip, isAdmin, roomID)
	peerDetails.AddPeer(publicKey3_VM3, ip, isAdmin, roomID)

	// // peerDetails.RemovePeer(publicKey, roomID)
	// // network.GetYggdrasilPeers()

	// network.InitializeNetwork("room-xyz-987")

	// gossipnetwork.InitializeNetwork("room-xyz-987")

	// Initialize the gossip network and get the instance
	gossipNet, err := gossipnetwork.InitializeGossipNetwork("room-xyz-987", 3000)
	if err != nil {
		log.Fatal("Failed to initialize gossip network:", err)
	}

	fmt.Println("Gossip network initialized successfully!")

	// Keep the program running so the listener continues
	fmt.Println("Gossip network listening on port 3000. Press Ctrl+C to exit.")

	// Example: Send a gossip message after 10 seconds
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("Sending first gossip message...")

		// Send a broadcast chat message to all peers
		gossipNet.GossipMessage("chat", "broadcast", "Hello everyone from node 1!", 0, roomID)

		// Send another message after 20 seconds
		time.Sleep(10 * time.Second)
		fmt.Println("Sending second gossip message...")
		gossipNet.GossipMessage("system", "broadcast", "Node 1 is still alive!", 0, roomID)

		// Refresh peer health every 2 minutes to prevent them from being marked dead
		go func() {
			ticker := time.NewTicker(2 * time.Minute)
			for range ticker.C {
				fmt.Println("Refreshing peer health...")
				gossipNet.RefreshPeerHealth()
			}
		}()

		// You can also send direct messages to specific peers if you know their IDs
		// Example (uncomment and replace with actual peer ID):
		// gossipNet.GossipMessage("chat", "direct", "Private message!", 16750950217577629675, roomID)
	}()

	// Keep the main function alive
	select {}
}
