package simulations

import (
	"blockchain-p2p-messenger/src/network"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)
func DigitalSignatureForgeryTest(roomID string, port uint64){
	// Construct a custom random private / public key pair (Recieved from running src/genkeys/genRandKeys)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	messageContent := "Digital Signature Forgery Test"
	
	// Construct our own message to send
	message := network.Message{
		PublicKey: hex.EncodeToString(pub),
		Message:   messageContent,
		RoomID:    roomID,
		Type: "chat",
		Timestamp: uint64(time.Now().Unix()),
	}
	// Sign the message with our random forged private / public key pair


	// Sign a message using the private key
	// message := []byte("Hello, this is a test message.")
	signature := ed25519.Sign(priv, []byte(message.Message))	
	message.DigitalSignature = hex.EncodeToString(signature)

	peers := peerDetails.GetPeersInRoom(roomID)


	for _, peer := range peers{
			// Dial peer
			fmt.Printf("Establishing connection with %s, %s.......\n", peer.IP, peer.PublicKey)
			address := net.JoinHostPort(peer.IP, fmt.Sprintf("%d", port))
			fmt.Println(address)

			conn, err := net.Dial("tcp", address)
			if err != nil {
				log.Printf("Error connecting to %s: %v\n", peer.IP, err)
				return
			}
			defer conn.Close()

			// Marshal message
			msgBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling message for %s: %v\n", peer.IP, err)
				return
			}

			// Send message
			_, err = conn.Write(msgBytes)
			if err != nil {
				log.Printf("Error sending message to %s: %v\n", peer.IP, err)
				return
			}

			// Read the response from the peer
			buffer := make([]byte, 1024) // Buffer to store incoming data
			n, err := conn.Read(buffer)
			if err != nil {
				log.Printf("Error reading response from %s: %v\n", peer.IP, err)
				return
			}

			// Print the response received from the peer
			response := string(buffer[:n])
			fmt.Printf("Response from %s: %s\n", peer.IP, response)
	}

}