package consensustests

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network"
	"blockchain-p2p-messenger/src/peerDetails"
)

// N = 4
func RunTest1(){

	// Setup Peers
	publicKey_VM1 := "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
	publicKey2_VM2 := "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
	publicKey3_VM3 := "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"

	roomID := "room-xyz-987"

	// Assume 1 Compromised Node
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID) // Compromised Node
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)

	network.InitializeNetwork(roomID, true)


}


