package consensustestsVM1

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
	"time"
)

var publicKey_VM1 = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var publicKey2_VM2 = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var publicKey4_VM4 = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"

// N = 4
// Using No Acks to send back (1 attacker)
// Every node except attacker node has to be pre run
// Attacker node(s) is ran via start message from stat collector, and it has to wait for this message to run
// Attacker node(s) then prepare for next run in which they wait for another start message
// Run is considered finished when 3 messages (blockchains) have been received, then ran again until desired limit (100 runs)

func RunConsensusTestControlVM1() {

	// Setup Peers
	roomID := "room-xyz-987"

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, true)
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(time.Second * 5)

	// Send 3 gossip messages with 1-second delays to allow other VMs to process each message
	fmt.Println("VM1: Starting consensus test - sending 3 messages...")

	// Message 1
	gossipNet.GossipMessage("chat", "broadcast", "Consensus Test Message 1", 1, roomID, "")
	fmt.Println("VM1: Sent message 1")
	time.Sleep(time.Second * 1)

	// Message 2
	gossipNet.GossipMessage("chat", "broadcast", "Consensus Test Message 2", 1, roomID, "")
	fmt.Println("VM1: Sent message 2")
	time.Sleep(time.Second * 1)

	// Message 3
	gossipNet.GossipMessage("chat", "broadcast", "Consensus Test Message 3", 1, roomID, "")
	fmt.Println("VM1: Sent message 3")
	fmt.Println("VM1: All 3 messages sent, waiting for consensus...")

}

// Case 1: 1 attacker (1 / 4 Attacker nodes)
func RunConsensusTestCase1VM1() {

}

// Case 2: 2 attackers (2 / 4 Attacker nodes)
func RunConsensusTestCase2VM1() {

}
