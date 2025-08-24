package consensustestsVM2

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
)

var	publicKey_VM1 = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var	publicKey2_VM2 = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var	publicKey3_VM3 = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var	publicKey4_VM4 = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"

var roomID = "room-xyz-987"

// N = 4
// VM2 will demonstrate Attack
func RunConsensusTestControlVM2() {

	// Setup Peers
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Set this to true to enable spam injection mode
	injectSpam := true

	gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, true, injectSpam)

}


// Case 1: 1 attacker (1 / 4 Attacker nodes)
func RunConsensusTestCase1VM2() {
	// Setup Peers

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Set this to true to enable spam injection mode
	injectSpam := true
	noAckBlockchainSave := false

	gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, noAckBlockchainSave, injectSpam)	

}