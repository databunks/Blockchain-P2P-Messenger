package gossip_test_VM4

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
	"log"
	"net"
)

var publicKey_VM1 string = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
var publicKey2_VM2 string = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 string = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var PublicKey4_VM4 string = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var isAdmin bool = false
var roomID string = "room-xyz-987" // mock room IDd

func main() {

	// Singular Attackers
	RunGossipTestControlVM4(false, 1)

	// A=1 F=2
	RunGossipTestControlVM4(false, 2)

	// A=1 F=3
	RunGossipTestControlVM4(false, 3)

	// A=1 F=1
	// RunGossipTestControlVM4(false, 1)

	// A=1 F=2
	// RunGossipTestControlVM1(false)

	// // A=1 F=3
	// RunGossipTestControlVM1(false)

	// // A=2 F=1
	// RunGossipTestControlVM1(true)

	// // A=2 F=2
	// RunGossipTestControlVM1(true)

	// // A=2 F=3
	// RunGossipTestControlVM1(true)

	// // A=3 F=1
	// RunGossipTestControlVM1(true)

	// // A=3 F=2
	// RunGossipTestControlVM1(true)

	// // A=3 F=3
	// RunGossipTestControlVM1(true)
}

func RunGossipTestControlVM4(runAsAttacker bool, fanout int) {
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), isAdmin, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), isAdmin, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), isAdmin, roomID)
	peerDetails.AddPeer(PublicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(PublicKey4_VM4), isAdmin, roomID)

	if runAsAttacker {
		// Send message to specific nodes
		network.SendMessage("Official group chat message!", roomID, 3000, "chat")
	} else {
		network.InitializeNetwork(roomID, true, true)

	}

}

func RunGossipTestImplementationVM4(runAsAttacker bool) {
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), isAdmin, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), isAdmin, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), isAdmin, roomID)
	peerDetails.AddPeer(PublicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(PublicKey4_VM4), isAdmin, roomID)

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, runAsAttacker, true, true, false)

	if err != nil {
		fmt.Println(err)
		return
	}

	// gossip message is sent from here to random nodes on network
	gossipNet.GossipMessage("chat", "broadcast", "I hope I don't get censored!", 0, roomID, "")

}

func ReceiveStartMessage(port int) {
	var yggdrasilNodeInfo = network.GetYggdrasilNodeInfo()

	address := fmt.Sprintf("[%s]:%d", yggdrasilNodeInfo.Address, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s for start message", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		defer conn.Close()
	}
}
