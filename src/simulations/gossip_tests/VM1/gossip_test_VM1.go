package gossiptestsVM1

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
)

// VM1 Configuration
var publicKey_VM1 string = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var publicKey2_VM2 string = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 string = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var publicKey4_VM4 string = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
var roomID string = "room-xyz-987"

// N = 4
// VM1 will demonstrate Gossip Test Control (Old Network)
func RunGossipTestControlVM1() {
	// Setup Peers
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), true, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	fmt.Printf("🚀 VM1: Initializing Gossip Test Control (Old Network)\n")
	fmt.Printf("   Room ID: %s\n", roomID)
	fmt.Printf("   Using old network.InitializeNetwork\n")

	// Use the old network initialization
	network.InitializeNetwork(roomID, true, true)

	// Send a test message to limited nodes (example: send to only 2 nodes)
	testMessage := "Hello from VM1 Control (Old Network)!"
	err := network.SendMessageToLimitedNodes(testMessage, roomID, 3000, "chat", 2)
	if err != nil {
		fmt.Printf("❌ Error sending limited message: %v\n", err)
	} else {
		fmt.Printf("✅ Sent limited message to 2 nodes: %s\n", testMessage)
	}

	fmt.Println("✅ VM1: Gossip Test Control (Old Network) initialized successfully")
}

// N = 4
// VM1 will demonstrate Gossip Test Case 1 (New Gossip Network)
func RunGossipTestCase1VM1() {
	// Setup Peers
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), true, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Network configuration parameters
	port := uint64(3000)
	toggleAttacker := false
	toggleBlockchain := true
	noAckBlockchainSave := true
	injectSpam := false
	disableAckSending := false
	forwardingFanout := 0 // Default: forward to all peers

	fmt.Printf("🚀 VM1: Initializing Gossip Test Case 1 (New Gossip Network)\n")
	fmt.Printf("   Room ID: %s\n", roomID)
	fmt.Printf("   Port: %d\n", port)
	fmt.Printf("   Blockchain: %t\n", toggleBlockchain)
	fmt.Printf("   No ACK Blockchain Save: %t\n", noAckBlockchainSave)
	fmt.Printf("   Inject Spam: %t\n", injectSpam)
	fmt.Printf("   Disable ACK Sending: %t\n", disableAckSending)
	fmt.Printf("   Forwarding Fanout: %d (0 = all peers)\n", forwardingFanout)

	gossipnetwork.InitializeGossipNetwork(roomID, port, toggleAttacker, toggleBlockchain, noAckBlockchainSave, injectSpam, disableAckSending, forwardingFanout)

	fmt.Println("✅ VM1: Gossip Test Case 1 (New Gossip Network) initialized successfully")
}
