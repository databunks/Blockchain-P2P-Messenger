package main

import (
	//"blockchain-p2p-messenger/src/genkeys"
	// "blockchain-p2p-messenger/src/simulations"
	// "blockchain-p2p-messenger/src/consensus"

	"blockchain-p2p-messenger/src/simulations/consensus_tests"
	// "crypto/ed25519"
	// "fmt"
)

func main() {

	consensustests.RunTest1()

	// network.StartYggdrasilServer()
	// publicKey_VM1 := "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
	// publicKey2_VM2 := "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
	// publicKey3_VM3 := "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
	// ip := "219:84b6:648e:9ca5:e124:49ed:42d2:e6a3" // fake IPv6 address
	// isAdmin := false
	// roomID := "room-xyz-987" // mock room IDd

	// peerDetails.AddPeer(publicKey_VM1, ip, isAdmin, roomID)
	// peerDetails.AddPeer(publicKey2_VM2, ip, isAdmin, roomID)
	// peerDetails.AddPeer(publicKey3_VM3, ip, isAdmin, roomID)

	// // peerDetails.RemovePeer(publicKey, roomID)
	// // network.GetYggdrasilPeers()

	// network.InitializeNetwork("room-xyz-987")

	//network.SendMessage("This is joe biden speaking", roomID, 3000, "chat")
	//network.ListenOnPort(3000)
	//

	//genkeys.GenerateKeys()
	// simulations.DigitalSignatureForgeryTest(roomID, 3000)
}
