package gossip_test

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
)

// N = 4, 7, 10

func RunGossipTest1(){
	publicKey_VM1 := "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
	publicKey2_VM2 := "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
	publicKey3_VM3 := "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
	PublicKey4_VM4 := "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
	isAdmin := false
	roomID := "room-xyz-987" // mock room IDd

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), isAdmin, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), isAdmin, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), isAdmin, roomID)
	peerDetails.AddPeer(PublicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(PublicKey4_VM4), isAdmin, roomID)
	

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false)

	if (err != nil){
		fmt.Println(err)
		return
	}

	// gossip message is sent from here to random nodes on network
	gossipNet.GossipMessage("chat", "broadcast", "I hope i don't get censored!", 0, roomID, "")



	// Start timer
	
	// output chat log and check for censored string


}


