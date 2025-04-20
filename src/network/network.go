package network

import (
	"blockchain-p2p-messenger/src/consensus"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/anthdm/hbbft"
	"github.com/joho/godotenv"
)

type Message struct {
	PublicKey      string `json:"public_key"`
	Message        string `json:"message"`
	DigitalSignature string `json:"digital_signature"`
	RoomID string `json:"room_id"`
}

type Peer struct {
	Remote    string  `json:"remote"`
	Up        bool    `json:"up"`
	Inbound   bool    `json:"inbound"`
	Address   string  `json:"address"`
	Key       string  `json:"key"`
	Port      int     `json:"port"`
	Priority  int     `json:"priority"`
	Cost      int     `json:"cost"`
	BytesRecv float64 `json:"bytes_recvd"`
	BytesSent float64 `json:"bytes_sent"`
	Uptime    float64 `json:"uptime"`
	Latency   float64 `json:"latency"`
}

type PeerList struct {
	Peers []Peer `json:"peers"`
}


type YggdrasilNodeInfo struct {
    BuildName       string `json:"build_name"`
    BuildVersion    string `json:"build_version"`
    Key             string `json:"key"`
    Address         string `json:"address"`
    RoutingEntries  int    `json:"routing_entries"`
    Subnet          string `json:"subnet"`
}

var HB *hbbft.HoneyBadger
var node *consensus.Node

func InitializeNetwork(roomID string) error{

	peers := peerDetails.GetPeersInRoom(roomID)

	for i := 0; i < 10; i++{

		peers = peerDetails.GetPeersInRoom(roomID)
		peersOnline, _, _ := GetYggdrasilPeers()

		peerSet := make(map[string]struct{})
		for _, p := range peers {
			peerSet[p.PublicKey] = struct{}{}
		}

		lenPeers := 0
		for _, p := range peersOnline {
			if _, ok := peerSet[p.Key]; ok {
				fmt.Println(p.Key, "is online")
				lenPeers++
			} else {
				fmt.Println(p.Key, "is offline")
			}
		}

		if lenPeers < 4 {
			fmt.Printf("Retrying.... (attempt %d)\n", i)
			time.Sleep(5 * time.Second)
		} else if (lenPeers == 4) {
			fmt.Printf("Initializing consensus with Yggdrasil peers: %v\n", peers)
			break
		} else if (i == 9){
			return fmt.Errorf("error: Not enough nodes have been connected (%d nodes connected)", i)
		}
	}


	


	// Link peers to consensus (public key is passed into node id)
	// (New node represents our local server)
	transport := make([]hbbft.Transport, 1)
	var peerIDs []uint64 

	// Translating public keys to uint34 IDs
	for _, peer := range peers {
		peerIDs = append(peerIDs, consensus.PublicKeyToID(peer.PublicKey))
	}


	node = consensus.NewNode(consensus.PublicKeyToID(GetYggdrasilNodeInfo().Key), transport[0], peerIDs)
	HB = node.HB
	
	HB.Start()

	return nil
}
// GetYggdrasilPeers returns a list of Yggdrasil peer IP addresses
func GetYggdrasilPeers() ([]Peer, []Peer, error) {
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getPeers").Output()

	if err != nil {
		log.Fatalf("Failed to run yggdrasilctl: %v", err)
	}

	var peerList PeerList
	if err := json.Unmarshal(out, &peerList); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Categorize peers
	inPeers := []Peer{}
	outPeers := []Peer{}

	for _, p := range peerList.Peers {
		if p.Inbound {
			inPeers = append(inPeers, p)
		} else {
			outPeers = append(outPeers, p)
		}
	}


	// log.Printf("Inbound Peers: %v", inPeers)
	// log.Printf("Outbound Peers: %v", outPeers)

	return inPeers, outPeers, err
}


func GetYggdrasilNodeInfo() YggdrasilNodeInfo{
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getSelf").Output()
	if err != nil {
		log.Fatalf("Failed to run yggdrasilctl: %v", err)
	}

	var yggdrasilNodeInfo YggdrasilNodeInfo
	if err := json.Unmarshal(out, &yggdrasilNodeInfo); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	return yggdrasilNodeInfo
}


func handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Connection received from %s", remoteAddr)

	// Read data from the connection
	buffer := make([]byte, 1024) // Buffer to store incoming data
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from %s: %v", remoteAddr, err)
		return
	}

	// Convert received data to a string and log it
	receivedMessage := string(buffer[:n])
	log.Printf("Received from %s: %s", remoteAddr, receivedMessage)




	// TODO: Timeout connection if nothing appears after 5 mins

	// TODO: verify public key through signature

	// TODO: if sender is not verified on allow list, reject connection 

	// TODO: if sender requests admin stuff and is not admin then reject (this would probably be in consensus)





	// Optionally, send a response back to the client
	responseMessage := "Message received!"
	conn.Write([]byte(responseMessage)) // Send response to the client
}

func ListenOnPort(port int) {
	var yggdrasilNodeInfo = GetYggdrasilNodeInfo()

	address := fmt.Sprintf("[%s]:%d", yggdrasilNodeInfo.Address, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		go handleConnection(conn)
	}
}



// SignMessage signs a message with the private key and returns the signature.
func SignMessage(message []byte) ([]byte, error) {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Access a specific environment variable
	hexPrivateKey := os.Getenv("PRIVATE_KEY")

	// Convert the hex string to bytes
	privateKeyBytes, err := hex.DecodeString(hexPrivateKey)
	if err != nil {
		log.Fatalf("Failed to decode hex string: %v", err)
	}

	hexPrivateKey = ""

	// Convert the byte slice to an ed25519.PrivateKey
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		log.Fatalf("Invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)

	// Sign the message using the private key
	signature, err := privateKey.Sign(rand.Reader, message, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %v", err)
	}
	return signature, nil
}

// sendMessage creates a Message struct, signs it, and returns the message with the digital signature
func sendMessage(publicKey string, messageContent string, roomID string) (Message, error) {
	// Create the message struct
	message := Message{
		PublicKey: publicKey,
		Message:   messageContent,
		RoomID:    roomID, 
	}

	// Sign the message using the SignMessage function
	signature, err := SignMessage([]byte(messageContent))
	if err != nil {
		return Message{}, fmt.Errorf("failed to sign message: %v", err)
	}

	// Add the signature to the message
	message.DigitalSignature = hex.EncodeToString(signature)

	// Return the message with the signature
	return message, nil
}


func StartYggdrasilServer() error {

	// inPeers, _, err := GetYggdrasilPeers()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	ListenOnPort(3000)

	return nil
}





