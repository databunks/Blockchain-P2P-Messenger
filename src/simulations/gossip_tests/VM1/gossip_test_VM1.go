package gossiptestsVM1

import (
	"blockchain-p2p-messenger/src/derivationFunctions"
	"blockchain-p2p-messenger/src/network"
	gossipnetwork "blockchain-p2p-messenger/src/network_gossip"
	"blockchain-p2p-messenger/src/peerDetails"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// VM1 Configuration
var publicKey_VM1 string = "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0"
var publicKey2_VM2 string = "927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285"
var publicKey3_VM3 string = "9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb"
var publicKey4_VM4 string = "0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd"
var roomID string = "room-xyz-987"

// Global gossip network instance
var gossipNet *gossipnetwork.GossipNetwork

// N = 4
// VM1 will demonstrate Gossip Test Control (Old Network)
// VM1 is the attacker VM which will selectively censor
func RunGossipTestControlVM1() {
	// Setup Peers
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), true, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	fmt.Printf("🚀 VM1: Initializing Gossip Test Control (Old Network)\n")
	fmt.Printf("   Room ID: %s\n", roomID)
	fmt.Printf("   Using old network.InitializeNetwork\n")

	// Start command listener in a goroutine
	go startCommandListener()

	// // Use the old network initialization
	// network.InitializeNetwork(roomID, true, true)

	// Send a test message to limited nodes (example: send to only 2 nodes)
	testMessage := "Official group chat message!"
	err := network.SendMessage(testMessage, roomID, 3000, "chat")
	if err != nil {
		fmt.Printf("❌ Error sending limited message: %v\n", err)
	} else {
		fmt.Printf("✅ Sent limited message to 2 nodes: %s\n", testMessage)
	}

	fmt.Println("✅ VM1: Gossip Test Control (Old Network) initialized successfully")
}

// N = 4
// VM1 will demonstrate Gossip Test Case 1 (New Gossip Network)
func RunGossipTestCaseVM1() {
	// Setup Peers
	peerDetails.AddPeer(publicKey_VM1, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey_VM1), true, roomID)
	peerDetails.AddPeer(publicKey2_VM2, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey2_VM2), false, roomID)
	peerDetails.AddPeer(publicKey3_VM3, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey3_VM3), false, roomID)
	peerDetails.AddPeer(publicKey4_VM4, derivationFunctions.DeriveIPAddressFromPublicKey(publicKey4_VM4), false, roomID)

	// Network configuration parameters
	port := uint64(3000)
	toggleAttacker := false
	toggleBlockchain := true
	noAckBlockchainSave := true
	injectSpam := false
	disableAckSending := false
	forwardingFanout := 3

	fmt.Printf("🚀 VM1: Initializing Gossip Test Case 1 (New Gossip Network)\n")
	fmt.Printf("   Room ID: %s\n", roomID)
	fmt.Printf("   Port: %d\n", port)
	fmt.Printf("   Blockchain: %t\n", toggleBlockchain)
	fmt.Printf("   No ACK Blockchain Save: %t\n", noAckBlockchainSave)
	fmt.Printf("   Inject Spam: %t\n", injectSpam)
	fmt.Printf("   Disable ACK Sending: %t\n", disableAckSending)
	fmt.Printf("   Forwarding Fanout: %d (0 = all peers)\n", forwardingFanout)

	// Initialize gossip network
	var err error
	gossipNet, err = gossipnetwork.InitializeGossipNetwork(roomID, port, toggleAttacker, toggleBlockchain, noAckBlockchainSave, injectSpam, disableAckSending, forwardingFanout)
	if err != nil {
		fmt.Printf("❌ Error initializing gossip network: %v\n", err)
		return
	}

	// Start command listener in a goroutine
	go startCommandListener()

	fmt.Println("✅ VM1: Gossip Test Case 1 (New Gossip Network) initialized successfully")
}

// startCommandListener listens for commands from the stat collector
func startCommandListener() {
	listener, err := net.Listen("tcp", "localhost:3001")
	if err != nil {
		fmt.Printf("❌ Error starting command listener: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("🎧 VM1: Command listener started on localhost:3001")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("❌ Error accepting connection: %v\n", err)
			continue
		}

		go handleCommandConnection(conn)
	}
}

// handleCommandConnection processes incoming commands
func handleCommandConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("❌ Error reading command: %v\n", err)
		return
	}

	command := strings.TrimSpace(string(buffer[:n]))
	fmt.Printf("📨 VM1: Received command: %s\n", command)

	if strings.HasPrefix(command, "SEND_LIMITED_MESSAGE:") {
		parts := strings.Split(command, ":")
		if len(parts) == 2 {
			maxNodesStr := parts[1]
			maxNodes, err := strconv.Atoi(maxNodesStr)
			if err != nil {
				fmt.Printf("❌ Error parsing maxNodes: %v\n", err)
				return
			}

			fmt.Printf("🚀 VM1: Sending limited message to %d nodes\n", maxNodes)
			testMessage := "Official group chat message!"
			err = network.SendMessageToLimitedNodes(testMessage, roomID, 3000, "chat", maxNodes)
			if err != nil {
				fmt.Printf("❌ Error sending limited message: %v\n", err)
			} else {
				fmt.Printf("✅ VM1: Sent limited message to %d nodes: %s\n", maxNodes, testMessage)
			}
		}
	} else if strings.HasPrefix(command, "SEND_GOSSIP_MESSAGE:") {
		parts := strings.Split(command, ":")
		if len(parts) == 2 {
			message := parts[1]
			if message == "" {
				message = "Official group chat message!"
			}

			fmt.Printf("🚀 VM1: Sending gossip message: %s\n", message)

			// Send the gossip message using the global gossip network instance
			if gossipNet != nil {
				gossipNet.GossipMessage("chat", "broadcast", message, 0, roomID, "")
				fmt.Printf("📤 VM1: Gossip message sent: %s\n", message)
			} else {
				fmt.Printf("❌ Error: Gossip network not initialized\n")
			}
		}
	}
}
