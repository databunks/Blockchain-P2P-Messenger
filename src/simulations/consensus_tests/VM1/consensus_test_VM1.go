package consensustestsVM1

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var publicKey_VM1 = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var publicKey2_VM2 = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var publicKey4_VM4 = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"

// Global variables to store network info
var globalGossipNet *gossipnetwork.GossipNetwork
var globalRoomID string

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

	// Set this to true to enable spam injection mode
	injectSpam := false

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, true, injectSpam)
	if err != nil {
		fmt.Println(err)
	}

	// Store the gossip network for later use
	globalGossipNet = gossipNet
	globalRoomID = roomID

	fmt.Println("VM1: Network initialized, waiting for start gossiping command...")

	// Start listening for commands from stat collector
	go listenForCommands()

}

// Case 1: 1 attacker (1 / 4 Attacker nodes)
func RunConsensusTestCase1VM1() {
	// Setup Peers
	roomID := "room-xyz-987"

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Set this to true to enable spam injection mode
	injectSpam := false
	noAckBlockchainSave := false

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, noAckBlockchainSave, injectSpam)
	if err != nil {
		fmt.Println(err)
	}

	// Store the gossip network for later use
	globalGossipNet = gossipNet
	globalRoomID = roomID

	fmt.Println("VM1: Network initialized, waiting for start gossiping command...")

	// Start listening for commands from stat collector
	go listenForCommands()

}

// Case 2: 2 attackers (2 / 4 Attacker nodes)
func RunConsensusTestCase2VM1() {
	// Setup Peers
	roomID := "room-xyz-987"

	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), false, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Set this to true to enable spam injection mode
	injectSpam := false
	noAckBlockchainSave := false

	gossipNet, err := gossipnetwork.InitializeGossipNetwork(roomID, 3000, false, true, noAckBlockchainSave, injectSpam)
	if err != nil {
		fmt.Println(err)
	}

	// Store the gossip network for later use
	globalGossipNet = gossipNet
	globalRoomID = roomID

	fmt.Println("VM1: Network initialized, waiting for start gossiping command...")

	// Start listening for commands from stat collector
	go listenForCommands()
}

// listenForCommands listens for commands from the stat collector
func listenForCommands() {
	listener, err := net.Listen("tcp", ":3001")
	if err != nil {
		fmt.Printf("VM1: Failed to start command listener: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("VM1: Command listener started on port 3000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("VM1: Failed to accept connection: %v\n", err)
			continue
		}

		go handleCommand(conn)
	}
}

// handleCommand processes commands from the stat collector
func handleCommand(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("VM1: Failed to read command: %v\n", err)
		return
	}

	var command map[string]interface{}
	err = json.Unmarshal(buffer[:n], &command)
	if err != nil {
		fmt.Printf("VM1: Failed to parse command: %v\n", err)
		return
	}

	if commandType, exists := command["type"]; exists && commandType == "start_gossiping" {
		runNumber := int(command["run"].(float64))
		fmt.Printf("VM1: Received start gossiping command for run %d\n", runNumber)

		// Send acknowledgment
		response := fmt.Sprintf("VM1 starting gossip for run %d", runNumber)
		conn.Write([]byte(response))

		// Execute the gossip sequence
		go executeGossipSequence(runNumber)
	} else if commandType == "clear_blockchain" {
		runNumber := int(command["run"].(float64))
		fmt.Printf("VM1: 🚨 RECEIVED CLEAR BLOCKCHAIN COMMAND for run %d\n", runNumber)
		fmt.Printf("VM1: ⏰ Current time: %s\n", time.Now().Format("15:04:05.000"))

		// Clear VM1's own blockchain
		err := clearBlockchain()
		if err != nil {
			response := fmt.Sprintf("VM1 failed to clear blockchain: %v", err)
			conn.Write([]byte(response))
			return
		}

		// Send gossip message to all nodes to clear their blockchains
		fmt.Printf("VM1: 📢 Sending clear blockchain gossip to all nodes...\n")
		globalGossipNet.GossipMessage("clear_blockchain", "broadcast", fmt.Sprintf("Clear blockchain for run %d", runNumber), 1, globalRoomID, "")

		response := fmt.Sprintf("VM1 blockchain cleared and clear command gossiped to all nodes for run %d", runNumber)
		conn.Write([]byte(response))
	} else {
		fmt.Printf("VM1: Unknown command type: %v\n", commandType)
	}
}

// executeGossipSequence sends the 3 gossip messages with delays
func executeGossipSequence(runNumber int) {
	if globalGossipNet == nil {
		fmt.Println("VM1: Error: Gossip network not initialized")
		return
	}

	fmt.Printf("VM1: Executing gossip sequence for run %d\n", runNumber)

	// Message 1
	globalGossipNet.GossipMessage("chat", "broadcast", fmt.Sprintf("Consensus Test Message 1 (Run %d)", runNumber), 1, globalRoomID, "")
	fmt.Println("VM1: Sent message 1")
	time.Sleep(time.Second * 3) // 3 second delay for better consensus

	// Message 2
	globalGossipNet.GossipMessage("chat", "broadcast", fmt.Sprintf("Consensus Test Message 2 (Run %d)", runNumber), 1, globalRoomID, "")
	fmt.Println("VM1: Sent message 2")
	time.Sleep(time.Second * 3) // 3 second delay for better consensus

	// Message 3
	globalGossipNet.GossipMessage("chat", "broadcast", fmt.Sprintf("Consensus Test Message 3 (Run %d)", runNumber), 1, globalRoomID, "")
	fmt.Println("VM1: Sent message 3")
	fmt.Printf("VM1: All 3 messages sent for run %d, waiting for consensus...\n", runNumber)
}

// clearBlockchain clears the blockchain file for the current room
func clearBlockchain() error {
	blockchainPath := fmt.Sprintf("data/%s/blockchain.json", globalRoomID)

	fmt.Printf("VM1: 🧹 Clearing blockchain for room: %s\n", globalRoomID)

	// Read the current blockchain to preserve genesis and peer blocks
	blockchainData, err := os.ReadFile(blockchainPath)
	if err != nil {
		return fmt.Errorf("failed to read blockchain file: %v", err)
	}

	// Parse the blockchain to find and keep only genesis and peer blocks
	var blockchain []map[string]interface{}
	if err := json.Unmarshal(blockchainData, &blockchain); err != nil {
		return fmt.Errorf("failed to parse blockchain: %v", err)
	}

	fmt.Printf("VM1: 📖 Read blockchain data: %d total blocks\n", len(blockchain))

	// Find the last peer block (look for PEER_ADDED in reverse order)
	var preservedBlocks []map[string]interface{}
	lastPeerIndex := -1

	for i := len(blockchain) - 1; i >= 0; i-- {
		block := blockchain[i]
		if data, exists := block["data"]; exists {
			if dataStr, ok := data.(string); ok && strings.Contains(dataStr, "PEER_ADDED") {
				lastPeerIndex = i
				break
			}
		}
	}

	// Preserve blocks from index 0 to lastPeerIndex (genesis + all peers)
	if lastPeerIndex >= 0 {
		preservedBlocks = blockchain[:lastPeerIndex+1]
		fmt.Printf("VM1: 💾 Preserving %d blocks (genesis + peers up to index %d)\n", len(preservedBlocks), lastPeerIndex)
	} else {
		// If no peer blocks found, just keep genesis
		preservedBlocks = blockchain[:1]
		fmt.Printf("VM1: 💾 Preserving %d blocks (genesis only)\n", len(preservedBlocks))
	}

	// Write back the preserved blocks
	preservedData, err := json.MarshalIndent(preservedBlocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preserved blocks: %v", err)
	}

	fmt.Printf("VM1: 📝 Writing preserved blockchain: %d bytes\n", len(preservedData))

	err = os.WriteFile(blockchainPath, preservedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write preserved blockchain: %v", err)
	}

	fmt.Printf("VM1: ✅ Blockchain cleared - kept genesis and peer blocks, removed chat messages\n")
	return nil
}
