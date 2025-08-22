package network

import (
	"blockchain-p2p-messenger/src/consensus"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Message struct {
	PublicKey        string `json:"public_key"`
	Message          string `json:"message"`
	Type             string `json:"type"`
	RoomID           string `json:"room_id"`
	DigitalSignature string `json:"digital_signature"`
	Timestamp        uint64 `json:"timestamp"`
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
	BuildName      string `json:"build_name"`
	BuildVersion   string `json:"build_version"`
	Key            string `json:"key"`
	Address        string `json:"address"`
	RoutingEntries int    `json:"routing_entries"`
	Subnet         string `json:"subnet"`
}


var nodes []*consensus.Server

var peerIDs []uint64

var nodeIDs []uint64

var disableAuth bool

var isCensoringTest bool

var receivedCensoredMessage bool


func init() {
    gob.Register(&consensus.Transaction{})
}


func InitializeNetwork(roomID string, disableAuthToggle bool, isCensoringTestToggle bool) error{
	disableAuth = disableAuthToggle

	peers := peerDetails.GetPeersInRoom(roomID)

	for i := 0; i < 10; i++ {

		peersOnline, _ := GetYggdrasilPeers("unique")

		peers = peerDetails.GetPeersInRoom(roomID)

		peerSet := make(map[string]struct{})
		for _, p := range peersOnline {
			peerSet[p.Key] = struct{}{}
		}

		lenPeers := 0
		for _, p := range peers {
			if _, ok := peerSet[p.PublicKey]; ok {
				fmt.Println(p.PublicKey, "is online")
				lenPeers++
			} else {
				fmt.Println(p.PublicKey, "is offline")
			}
		}
		if lenPeers < 3 {
			fmt.Printf("Retrying Connection.... (attempt %d)\n", i)
			time.Sleep(5 * time.Second)
		} else if lenPeers >= 3 {
			fmt.Printf("Initializing consensus with Yggdrasil peers: %v\n", peers)
			break
		} else if i == 9 {
			return fmt.Errorf("error: Not enough nodes have been connected (%d nodes connected)", i)
		}
	}

	// Link peers to consensus (public key is passed into node id)
	// (New node represents our local server)



	// Adding current public key in
	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the ED25519 private key from the environment variable
	publicKeyHex := os.Getenv("PUBLIC_KEY")
	if publicKeyHex == "" {
		log.Fatal("ED25519_PUBLIC_KEY is not set in .env file")
	}


	peerIDs = append(peerIDs, PublicKeyToID(publicKeyHex))

	// Translating public keys to uint34 IDs
	for _, peer := range peers {
		peerIDs = append(peerIDs, PublicKeyToID(peer.PublicKey))
	}

	// Sort nodeIDs in ascending order
    sort.Slice(peerIDs, func(i, j int) bool {
        return peerIDs[i] < peerIDs[j]
    })

	

	for i, id := range peerIDs {
        fmt.Printf("Node original ID %d assigned index ID %d\n", id, i)
		nodeIDs = append(nodeIDs, uint64(i))
    }

	
	go ListenOnPort(3000)

	nodes = consensus.InitializeConsensus(len(nodeIDs), nodeIDs)

	time.Sleep(time.Minute * 1)

	return nil
}




// Function to check if an item is not in a list
func isNotInList(item Peer, list []Peer) bool {
	for _, v := range list {
		if v == item {
			return false // Item found, it's in the list
		}
	}
	return true // Item not found, it's not in the list
}

// GetYggdrasilPeers returns a list of Yggdrasil peer IP addresses
func GetYggdrasilPeers(mode string) ([]Peer, error) {
	out, err := exec.Command("sudo", "yggdrasilctl", "-json", "getPeers").Output()

	if err != nil {
		log.Fatalf("Failed to run yggdrasilctl: %v", err)
	}

	var peerList PeerList
	if err := json.Unmarshal(out, &peerList); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	switch mode {
	case "inbound":
		inPeers := []Peer{}
		for _, p := range peerList.Peers {
			if p.Inbound {
				inPeers = append(inPeers, p)
			}
		}
		return inPeers, err

	case "outbound":
		outPeers := []Peer{}
		for _, p := range peerList.Peers {
			if !p.Inbound {
				outPeers = append(outPeers, p)
			}
		}
		return outPeers, err

	case "unique":
		uniquePeers := []Peer{}
		for _, p := range peerList.Peers {
			if isNotInList(p, uniquePeers) {
				uniquePeers = append(uniquePeers, p)
			}
		}
		return uniquePeers, err

	}

	return nil, nil
}

func GetYggdrasilNodeInfo() YggdrasilNodeInfo {
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
	// receivedMessage := string(buffer[:n])
	// log.Printf("Received from %s: %s", remoteAddr, receivedMessage)

	// Unmarshal the message into the Message struct
	var message Message
	err = json.Unmarshal(buffer[:n], &message)
	if err != nil {
		log.Printf("Error unmarshaling message: %v\n", err)
		return
	}
	message.Timestamp = uint64(time.Now().Unix())

	// First we check if public key is in the room
	peers := peerDetails.GetPeersInRoom(message.RoomID)
	for _, peer := range peers {
		log.Printf("Peers Public Key: %s\nMessage Public Key: %s\n", peer.PublicKey, message.PublicKey)
		if message.PublicKey == peer.PublicKey || disableAuth {
			// Test cases: Intrusion rate, verification speed, timing
			// Proper authorization
			// Confidentialty: E2EE, tampering,
			// Integrity of blockchain

			if (!disableAuth){
				// Validate the digital signature
				if !VerifyMessageSignature([]byte(message.Message), message.DigitalSignature, peer.PublicKey) {
					log.Printf("Invalid signature for message from %s\nMessage: %s\nSignature:%s", message.PublicKey, message.Message, message.DigitalSignature)

					responseMessage := "Invalid Digital Signature!!"
					conn.Write([]byte(responseMessage)) // Send response to the client
					return
				}
			}
			
			log.Printf("Authenticated!")

			switch message.Type {
			case "chat":
				if (isCensoringTest){
					if message.Message == "Official group chat message!"{
						SendMessageToStatCollector("Recieved Censored Message", message.RoomID, 3001)
					}
				}


				// // Doesent store message content (apart from sender, type, digital signature and timestamp)
				// tx := consensus.NewTransaction(message.PublicKey, "chat", message.DigitalSignature, message.Timestamp, message.RoomID)
				// tx1 := consensus.NewTransaction(message.PublicKey, "chatter", message.DigitalSignature, message.Timestamp, message.RoomID)
				// //tx2 := consensus.NewTransaction(message.PublicKey, "chat", message.DigitalSignature, message.Timestamp, message.RoomID)
				// //tx3 := consensus.NewTransaction(message.PublicKey, "chat", message.DigitalSignature, message.Timestamp, message.RoomID)

				// // var nodeIndex int

				// // for i, node := range nodes {
				// // 	if node.ID == PublicKeyToNodeID(message.PublicKey){
				// // 		nodeIndex = i
				// // 	}
				// // }

				// // nodes[nodeIndex].HB.AddTransaction(tx)

				// nodes[3].HB.AddTransaction(tx1)
				// nodes[0].HB.AddTransaction(tx)
				// nodes[1].HB.AddTransaction(tx)
				// nodes[2].HB.AddTransaction(tx)
				

				


				// // TODO: Adding them as transactions and syncing with consensus

				
				// fmt.Println("Added Message as Transaction")
			}

			
			return
		}
	}

	responseMessage := "Public Key Not Found In Allow List!"
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
// func SignMessage(message []byte) ([]byte, error) {

// 	// Load environment variables from .env file
// 	err := godotenv.Load("../keydetails.env")
// 	if err != nil {
// 		log.Fatalf("Error loading .env file: %v", err)
// 	}

// 	// Access a specific environment variable
// 	hexPrivateKey := os.Getenv("PRIVATE_KEY")

// 	// Convert the hex string to bytes
// 	privateKeyBytes, err := hex.DecodeString(hexPrivateKey)
// 	if err != nil {
// 		log.Fatalf("Failed to decode hex string: %v", err)
// 	}

// 	hexPrivateKey = ""

// 	// Convert the byte slice to an ed25519.PrivateKey
// 	if len(privateKeyBytes) != ed25519.PrivateKeySize {
// 		log.Fatalf("Invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes))
// 	}

// 	privateKey := ed25519.PrivateKey(privateKeyBytes)

// 	signature := privateKey.Sign(message)

// 	return signature, nil
// }

func SignMessage(message []byte) []byte {
	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the ED25519 private key from the environment variable
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("ED25519_PRIVATE_KEY is not set in .env file")
	}

	// Convert the hex string to bytes
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatal("Failed to decode private key:", err)
	}

	// Ensure the private key is the correct length (32 bytes for ED25519)
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		log.Fatal("Invalid private key size, expected 32 bytes")
	}

	// Generate the public key from the private key
	privateKey := ed25519.PrivateKey(privateKeyBytes)

	// Sign a message using the private key
	// message := []byte("Hello, this is a test message.")
	signature := ed25519.Sign(privateKey, message)

	return signature
}

// VerifyMessageSignature checks if the message was signed correctly using the sender's public key
func VerifyMessageSignature(messageContent []byte, signatureHex string, publicKeyHex string) bool {
	// Decode the hex-encoded public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		log.Printf("Error decoding public key: %v", err)
		return false
	}

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		log.Printf("Error decoding signature: %v", err)
		return false
	}

	// Check signature validity using ed25519
	isValid := ed25519.Verify(ed25519.PublicKey(publicKeyBytes), messageContent, signatureBytes)

	if !isValid {
		log.Println("Signature verification failed")
	}

	return isValid
}

func SendCustomMessage(messageContent string, publicKeyHex string, roomID string, port uint64, typeofmessage string) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	var wg sync.WaitGroup

	for _, peer := range peers {
		// ignore if peer in blockchain

		wg.Add(1)
		go func(peer peerDetails.Peer) {
			defer wg.Done()

			// Create the message struct
			message := Message{
				PublicKey: publicKeyHex,
				Message:   messageContent,
				RoomID:    roomID,
				Type:      typeofmessage,
				Timestamp: uint64(time.Now().Unix()),
			}

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

		}(peer) // Pass peer as an argument to avoid closure capture issues
	}

	wg.Wait() // Wait for all goroutines to finish
	return nil
}

func SendMessageToStatCollector(messageContent string, roomID string, port int){

	var statCollectorPublicKey string = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"

	peers := peerDetails.GetPeersInRoom(roomID)

	for _, peer := range peers{

		if peer.PublicKey == statCollectorPublicKey {
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
			msgBytes, err := json.Marshal(messageContent)
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


}

func SendMessage(messageContent string, roomID string, port uint64, typeofmessage string) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	// Load environment variables from .env file
	err := godotenv.Load("../keydetails.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the ED25519 private key from the environment variable
	publicKeyHex := os.Getenv("PUBLIC_KEY")
	if publicKeyHex == "" {
		log.Fatal("ED25519_PUBLIC_KEY is not set in .env file")
	}

	var wg sync.WaitGroup

	for _, peer := range peers {
		// ignore if peer in blockchain

		wg.Add(1)
		go func(peer peerDetails.Peer) {
			defer wg.Done()

			// Create the message struct
			message := Message{
				PublicKey: publicKeyHex,
				Message:   messageContent,
				RoomID:    roomID,
				Type:      typeofmessage,
				Timestamp: uint64(time.Now().Unix()),
			}

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

		}(peer) // Pass peer as an argument to avoid closure capture issues
	}

	wg.Wait() // Wait for all goroutines to finish
	return nil
}

func StartYggdrasilServer() error {

	// inPeers, _, err := GetYggdrasilPeers()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	ListenOnPort(3000)

	return nil
}



// PublicKeyToID converts a hex-encoded ed25519 public key string into a deterministic uint64 ID.
func PublicKeyToID(hexStr string) uint64 {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("invalid hex string: %v", err)
	}

	if len(bytes) != ed25519.PublicKeySize {
		log.Fatalf("invalid key length: expected %d, got %d", ed25519.PublicKeySize, len(bytes))
	}

	hash := sha256.Sum256(bytes)
	return binary.LittleEndian.Uint64(hash[:8])
}


func PublicKeyToNodeID(hexStr string) uint64 {

	ID := PublicKeyToID(hexStr)

	for i, peerID := range peerIDs {
		if ID == peerID {
			return uint64(i)
		}
	}

	return uint64(0)
}