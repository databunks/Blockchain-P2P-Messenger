package main

import (
	"blockchain-p2p-messenger/src/peerDetails"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var reachabilityCount = 0
var lastReceivedMessageTime time.Time
var timestamp_arrived []uint64

// Consensus testing variables
var currentRun int = 0
var totalRuns int = 50
var messagesThisRun int = 0
var runStartTime time.Time
var consensusMutex sync.Mutex
var isTestRunning bool = false
var attackerNodes []string // List of attacker node public keys

// Enhanced tracking for consensus integrity
var nodeBlockchains map[string][]string // nodeID -> []blockchainData for current run
var consensusIntegrityResults []float64 // Store integrity scores for each run
var attackSuccessRates []float64        // Store ASR for each run
var totalLatencies []int64              // Store latencies for each run
var honestNodes []string                // List of honest node public keys

func main() {
	// Initialize consensus testing
	initializeConsensusTest()

	go ListenOnPort(":3002")

	fmt.Println("Consensus Stat Collector Started!")
	fmt.Printf("Target: %d runs with 12 blockchains per run (3 messages from VM1)\n", totalRuns)

	// Start the first run
	startNewRun()

	// Wait for all runs to complete
	waitForTestCompletion()

	fmt.Println("=== CONSENSUS TEST COMPLETED ===")
	fmt.Printf("Total Runs: %d\n", currentRun)
	fmt.Printf("Reachability Count: %d\n", reachabilityCount)
	if len(timestamp_arrived) > 0 {
		fmt.Printf("Latency: %s\n", lastReceivedMessageTime.UnixMilli())
	}

	// Display final consensus results
	displayFinalResults()

	// Save results to CSV file
	saveResultsToCSV()
}

// initializeConsensusTest sets up the consensus testing environment
func initializeConsensusTest() {
	// Define attacker nodes
	attackerNodes = []string{

		"0000040cd8e7f870ff1146e03589b988d82aedb6464c5085a9aba945e60c4fcd",
	}

	// Define honest nodes
	honestNodes = []string{
		"0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0",
		"927c78b7fa731c2b2f642a1de2fb3318f70bbb142465a75a8802a90e1a526285",
		"9356e1f92f5adff2ab05115d54aff4b8c756d604704b5ddd71ff320f2d5aeecb",
	}

	// Initialize tracking maps
	nodeBlockchains = make(map[string][]string)
	consensusIntegrityResults = make([]float64, 0)
	attackSuccessRates = make([]float64, 0)
	totalLatencies = make([]int64, 0)

	fmt.Println("=== CONSENSUS TEST INITIALIZATION ===")
	fmt.Printf("Total Nodes: 4 (2 Honest + 2 Sybil)\n")
	fmt.Printf("Honest Nodes: %d\n", len(honestNodes))
	for i, key := range honestNodes {
		fmt.Printf("  Honest %d: %s\n", i+1, key[:16]+"...")
	}
	fmt.Printf("Attacker Nodes: %d\n", len(attackerNodes))
	for i, key := range attackerNodes {
		fmt.Printf("  Attacker %d: %s\n", i+1, key[:16]+"...")
	}
	fmt.Println("Expected: 3 messages from VM1 = 12 blockchains total per run")
	fmt.Println("VM1: Sends 3 messages with 1-second delays")
	fmt.Println("VM2, VM3, VM4: Process messages and send blockchains automatically")
	fmt.Println("All VMs: Send blockchains to stat collector via network_gossip.go")
}

// startNewRun begins a new consensus test run
func startNewRun() {
	consensusMutex.Lock()
	defer consensusMutex.Unlock()

	if currentRun >= totalRuns {
		fmt.Println("All runs completed!")
		return
	}

	currentRun++
	messagesThisRun = 0
	runStartTime = time.Now()
	isTestRunning = true

	// Reset blockchain tracking for new run
	nodeBlockchains = make(map[string][]string)

	fmt.Printf("\n=== STARTING RUN %d/%d ===\n", currentRun, totalRuns)
	fmt.Printf("Run started at: %s\n", runStartTime.Format("15:04:05"))
	fmt.Printf("Expected: 12 blockchains (3 messages from VM1 = 12 blockchains total)\n")
	fmt.Println("Sending start gossiping command to VM1...")

	// Send start gossiping command to VM1
	sendStartGossipingCommand()

}

// sendStartMessageToAttackerNodes notifies attacker nodes to begin their attack
func sendStartMessageToAttackerNodes() {
	fmt.Println("Sending start message to attacker nodes...")

	for _, attackerKey := range attackerNodes {
		// Find the attacker node in the room
		peers := peerDetails.GetPeersInRoom("room-xyz-987")
		for _, peer := range peers {
			if peer.PublicKey == attackerKey {
				fmt.Printf("Starting attacker node: %s...\n", attackerKey[:16]+"...")

				// Send actual start message to attacker node
				go sendStartMessageToNode(peer.IP, 3000, attackerKey)
				break
			}
		}
	}
}

// sendStartMessageToNode sends a start message to a specific node
func sendStartMessageToNode(nodeIP string, port int, nodeKey string) {
	address := net.JoinHostPort(nodeIP, fmt.Sprintf("%d", port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Failed to connect to attacker %s: %v\n", nodeKey[:16]+"...", err)
		return
	}
	defer conn.Close()

	// Create start message
	startMsg := map[string]interface{}{
		"type":      "start_attack",
		"run":       currentRun,
		"timestamp": time.Now().Unix(),
	}

	// Marshal and send
	msgBytes, err := json.Marshal(startMsg)
	if err != nil {
		fmt.Printf("Failed to marshal start message: %v\n", err)
		return
	}

	_, err = conn.Write(msgBytes)
	if err != nil {
		fmt.Printf("Failed to send start message to %s: %v\n", nodeKey[:16]+"...", err)
		return
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Failed to read response from %s: %v\n", nodeKey[:16]+"...", err)
		return
	}

	response := string(buffer[:n])
	fmt.Printf("Attacker %s response: %s\n", nodeKey[:16]+"...", response)
}

// isRunComplete checks if the current run has finished
func isRunComplete() bool {
	consensusMutex.Lock()
	defer consensusMutex.Unlock()

	// Check if we have enough messages or timeout
	timeoutReached := time.Since(runStartTime) > 60*time.Second // Increased timeout for 12 blockchains
	messagesReached := messagesThisRun >= 12                    // Expect 12 blockchains (3 per node × 4 nodes)

	if timeoutReached {
		fmt.Printf("Run %d: TIMEOUT reached (60s)\n", currentRun)
		return true
	}

	if messagesReached {
		fmt.Printf("Run %d: COMPLETED with %d/12 blockchains\n", currentRun, messagesThisRun)
		return true
	}

	return false
}

// waitForTestCompletion waits for all runs to finish
func waitForTestCompletion() {
	for currentRun < totalRuns {
		// Wait for current run to complete
		for isTestRunning && !isRunComplete() {
			time.Sleep(1 * time.Second)
		}

		if currentRun < totalRuns {
			// Start next run
			time.Sleep(2 * time.Second) // Brief pause between runs
			startNewRun()
		}
	}
}

// processMessageForConsensus handles messages for consensus testing
func processMessageForConsensus(msg string, nodeID string) {
	consensusMutex.Lock()
	defer consensusMutex.Unlock()

	if !isTestRunning {
		return
	}

	// Increment message count for current run
	messagesThisRun++

	// Safely truncate nodeID for display (avoid slice bounds error) - commented out for cleaner output
	// var displayID string
	// if len(nodeID) >= 16 {
	// 	displayID = nodeID[:16] + "..."
	// } else {
	// 	displayID = nodeID
	// }

	// fmt.Printf("Run %d: Blockchain %d/12 received from %s\n", currentRun, messagesThisRun, displayID)

	// Check if run is complete
	if messagesThisRun >= 12 {
		isTestRunning = false
		fmt.Printf("Run %d: All 12 blockchains received, assessing consensus integrity...\n", currentRun)

		// Assess consensus integrity for this run
		integrityScore := assessConsensusIntegrity()
		consensusIntegrityResults = append(consensusIntegrityResults, integrityScore)

		// Calculate attack success rate
		asr := calculateAttackSuccessRate()
		attackSuccessRates = append(attackSuccessRates, asr)

		fmt.Printf("Run %d: Consensus Integrity: %.2f%%, ASR: %.2f%%\n", currentRun, integrityScore*100, asr*100)

		// Wait a bit to ensure all blockchains are properly sent before clearing
		fmt.Printf("⏳ Waiting 5 seconds to ensure all blockchains are sent...\n")
		time.Sleep(5 * time.Second)

		// Clear blockchains on all VMs before next run
		fmt.Printf("🧹 Clearing blockchains on all VMs for next run...\n")
		clearBlockchainsOnAllVMs()

		// Wait 5 seconds for cleanup
		fmt.Printf("⏳ Waiting 5 seconds for blockchain cleanup...\n")
		time.Sleep(5 * time.Second)
	}
}

// assessConsensusIntegrity measures the percentage of honest nodes with identical blockchain state
func assessConsensusIntegrity() float64 {
	if len(honestNodes) == 0 {
		return 0.0
	}

	// Count honest nodes with blockchain data
	consistentNodes := 0
	totalHonestNodes := len(honestNodes)

	// Check if honest nodes have blockchain data and compare their contents
	for i, honestNode := range honestNodes {
		fmt.Printf("   🔍 Checking honest node: %s...\n", honestNode[:16]+"...")

		// Map honest node to the corresponding generic node ID
		genericNodeID := fmt.Sprintf("node_%d", i)
		fmt.Printf("   🔍 Looking for blockchain under: %s\n", genericNodeID)

		if blockchains, exists := nodeBlockchains[genericNodeID]; exists && len(blockchains) > 0 {
			fmt.Printf("   ✅ Found %d blockchain(s) for honest node\n", len(blockchains))

			// Get the first blockchain from this node
			nodeBlockchain := blockchains[0]

			// Compare with other honest nodes' blockchains
			isConsistent := true
			for j, otherHonestNode := range honestNodes {
				if i == j {
					continue // Skip self-comparison
				}

				// Map other honest node to its generic ID
				otherGenericNodeID := fmt.Sprintf("node_%d", j)
				fmt.Printf("   🔍 Comparing with honest node: %s (generic ID: %s)...\n", otherHonestNode[:16]+"...", otherGenericNodeID)

				if otherBlockchains, otherExists := nodeBlockchains[otherGenericNodeID]; otherExists && len(otherBlockchains) > 0 {
					fmt.Printf("   ✅ Found %d blockchain(s) for comparison node\n", len(otherBlockchains))

					otherBlockchain := otherBlockchains[0]

					// Compare blockchain contents
					if !compareBlockchains(nodeBlockchain, otherBlockchain) {
						fmt.Printf("   ❌ Blockchain comparison failed\n")
						isConsistent = false
						break
					} else {
						fmt.Printf("   ✅ Blockchain comparison succeeded\n")
					}
				} else {
					fmt.Printf("   ❌ No blockchain data found for comparison node\n")
					isConsistent = false
					break
				}
			}

			if isConsistent {
				consistentNodes++
				fmt.Printf("   ✅ Node marked as consistent\n")
			} else {
				fmt.Printf("   ❌ Node marked as inconsistent\n")
			}
		} else {
			fmt.Printf("   ❌ No blockchain data found for honest node\n")
		}
	}

	integrity := float64(consistentNodes) / float64(totalHonestNodes)

	// Debug logging
	fmt.Printf("🔍 Consensus Integrity Debug:\n")
	fmt.Printf("   - Total honest nodes: %d\n", totalHonestNodes)
	fmt.Printf("   - Nodes with blockchain data: %d\n", consistentNodes)
	fmt.Printf("   - Integrity score: %.2f%%\n", integrity*100)

	// Show the mapping between honest nodes and generic IDs
	fmt.Printf("   - Honest node to generic ID mapping:\n")
	for i, honestNode := range honestNodes {
		genericID := fmt.Sprintf("node_%d", i)
		fmt.Printf("     * %s... -> %s\n", honestNode[:16], genericID)
	}

	// Log what's in nodeBlockchains
	fmt.Printf("   - nodeBlockchains contents:\n")
	for nodeID, blockchains := range nodeBlockchains {
		fmt.Printf("     * %s: %d blockchains\n", nodeID, len(blockchains))
		if len(blockchains) > 0 {
			// Show a preview of the first blockchain
			firstBlock := blockchains[0]
			if len(firstBlock) > 100 {
				fmt.Printf("       Preview: %s...\n", firstBlock[:100])
			} else {
				fmt.Printf("       Preview: %s\n", firstBlock)
			}
		}
	}

	return integrity
}

// compareBlockchains compares two blockchain strings for consistency
func compareBlockchains(blockchain1, blockchain2 string) bool {
	// Parse blockchain data to extract actual block content
	blocks1 := extractBlocksFromBlockchain(blockchain1)
	blocks2 := extractBlocksFromBlockchain(blockchain2)

	// Compare number of blocks
	if len(blocks1) != len(blocks2) {
		fmt.Printf("   🔍 Block count mismatch: %d vs %d\n", len(blocks1), len(blocks2))
		return false
	}

	// Compare each block
	for i, block1 := range blocks1 {
		if i >= len(blocks2) {
			fmt.Printf("   🔍 Block %d missing in second blockchain\n", i)
			return false
		}

		block2 := blocks2[i]
		if !compareBlock(block1, block2) {
			fmt.Printf("   🔍 Block %d content mismatch\n", i)
			return false
		}
	}

	return true
}

// extractBlocksFromBlockchain extracts individual blocks from blockchain string in correct order
func extractBlocksFromBlockchain(blockchainStr string) []string {
	var blocks []string

	// Try to parse as JSON first to get proper ordering
	var blockchainData []map[string]interface{}
	if err := json.Unmarshal([]byte(blockchainStr), &blockchainData); err == nil {
		// Successfully parsed as JSON, extract blocks in index order
		for _, block := range blockchainData {
			if data, exists := block["data"]; exists {
				if dataStr, ok := data.(string); ok {
					// Check if it's a message block (not genesis or peer blocks)
					if strings.HasPrefix(dataStr, "CHAT_MSG{") || strings.HasPrefix(dataStr, "SPAM_MSG{") {
						blocks = append(blocks, dataStr)
					}
				}
			}
		}
		return blocks
	}

	// Fallback: if JSON parsing fails, use string splitting (less accurate)
	fmt.Printf("   ⚠️  JSON parsing failed, using fallback string splitting\n")

	// Extract CHAT_MSG blocks
	chatParts := strings.Split(blockchainStr, "CHAT_MSG{")
	for i, part := range chatParts {
		if i == 0 {
			continue // Skip first part (before first CHAT_MSG)
		}

		// Find the closing brace
		if endIndex := strings.Index(part, "}"); endIndex != -1 {
			block := "CHAT_MSG{" + part[:endIndex] + "}"
			blocks = append(blocks, block)
		}
	}

	// Extract SPAM_MSG blocks
	spamParts := strings.Split(blockchainStr, "SPAM_MSG{")
	for i, part := range spamParts {
		if i == 0 {
			continue // Skip first part (before first SPAM_MSG)
		}

		// Find the closing brace
		if endIndex := strings.Index(part, "}"); endIndex != -1 {
			block := "SPAM_MSG{" + part[:endIndex] + "}"
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// compareBlock compares two individual blocks, ignoring variable fields like timestamps
func compareBlock(block1, block2 string) bool {
	// Parse the block content to extract only essential fields
	content1 := extractBlockContent(block1)
	content2 := extractBlockContent(block2)

	// Compare only the essential content (sender, type, data)
	return content1 == content2
}

// extractBlockContent extracts only the essential content from a block, ignoring timestamps/signatures
func extractBlockContent(block string) string {
	// Remove timestamp and signature from comparison
	// Format: CHAT_MSG{Sender: X, Type: Y, Data: Z, Timestamp: T, Signature: S}
	// Format: SPAM_MSG{Attacker: X, Data: Y, Timestamp: T, Hash: H, Signature: S}

	// Find the start of the block
	if !strings.HasPrefix(block, "CHAT_MSG{") && !strings.HasPrefix(block, "SPAM_MSG{") {
		return block // Return as-is if not a recognized message type
	}

	// Extract the content between the message type and the first Timestamp
	parts := strings.Split(block, "Timestamp:")
	if len(parts) < 2 {
		return block // Return as-is if can't parse
	}

	// Take everything before "Timestamp:" and add closing brace
	essentialContent := parts[0] + "}"

	// Also remove any Signature field if present
	if strings.Contains(essentialContent, "Signature:") {
		sigParts := strings.Split(essentialContent, "Signature:")
		if len(sigParts) >= 2 {
			// Remove everything after "Signature:" and add closing brace
			essentialContent = sigParts[0] + "}"
		}
	}

	return essentialContent
}

// calculateAttackSuccessRate measures the success rate of attacker attacks
func calculateAttackSuccessRate() float64 {
	if len(attackerNodes) == 0 {
		return 0.0
	}

	// Count attacker nodes that successfully influenced consensus
	successfulAttacks := 0
	totalAttackerNodes := len(attackerNodes)

	// Check if attacker nodes have blockchain data (indicating successful participation)
	for _, attackerNode := range attackerNodes {
		if blockchains, exists := nodeBlockchains[attackerNode]; exists && len(blockchains) > 0 {
			successfulAttacks++
		}
	}

	asr := float64(successfulAttacks) / float64(totalAttackerNodes)
	return asr
}

// extractNodeIDFromMessage extracts the node ID from a message
func extractNodeIDFromMessage(msg string) string {
	// For "Message Reached To Peer" messages, extract the peer ID
	if strings.HasPrefix(msg, "Message Reached To Peer ") {
		parts := strings.Split(msg, "Message Reached To Peer ")
		if len(parts) > 1 {
			return parts[1]
		}
	}

	// Default to a placeholder if we can't extract
	return "unknown_node"
}

func ListenOnPort(port string) {
	// Listen on the specified port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Server is listening on port %s...\n", port)

	// Accept incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		fmt.Printf("New connection established: %v\n", conn.RemoteAddr())

		// Handle connection
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	// Close connection
	defer conn.Close()

	for {
		defer conn.Close()

		// Read message sent by client - increased buffer size for blockchain data
		buffer := make([]byte, 65536) // 64KB buffer for large blockchain data
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			return
		}

		// Try to unmarshal as blockchain message first
		var blockchainMsg map[string]interface{}
		err = json.Unmarshal(buffer[:n], &blockchainMsg)
		if err == nil {
			// Check if this is a blockchain message
			if msgType, exists := blockchainMsg["type"]; exists && msgType == "blockchain_data" {
				handleBlockchainMessage(blockchainMsg)
			} else {
				// Fall back to string message handling
				handleStringMessage(string(buffer[:n]))
			}
		} else {
			// Try to unmarshal as simple string message (legacy format)
			var msg string
			err = json.Unmarshal(buffer[:n], &msg)
			if err != nil {
				fmt.Println("Error unmarshaling message:", err)
				return
			}
			handleStringMessage(msg)
		}

		// Response back to client
		_, err = conn.Write([]byte("Message received!\n"))
		if err != nil {
			fmt.Printf("Error writing to client: %v\n", err)
		}
	}
}

// handleBlockchainMessage processes blockchain data messages
func handleBlockchainMessage(blockchainMsg map[string]interface{}) {
	// Extract blockchain data for logging (only data is used for spam detection)
	// var roomID string
	// var timestamp interface{}
	var data interface{}

	// if r, exists := blockchainMsg["room_id"]; exists {
	// 	roomID = fmt.Sprintf("%v", r)
	// }

	// if ts, exists := blockchainMsg["timestamp"]; exists {
	// 	timestamp = ts
	// }

	if d, exists := blockchainMsg["data"]; exists {
		data = d
	}

	// Enhanced logging for blockchain reception (commented out for cleaner output)
	// fmt.Printf("\n🔗 BLOCKCHAIN RECEIVED [Run %d] 🔗\n", currentRun)
	// fmt.Printf("📅 Time: %s\n", time.Now().Format("15:04:05.000"))
	// fmt.Printf("🏠 Room: %s\n", roomID)
	// fmt.Printf("⏰ Timestamp: %v\n", timestamp)

	// Log blockchain content summary (commented out for cleaner output)
	// if dataStr, ok := data.(string); ok {
	// 	// Count the number of blocks in the blockchain
	// 	blockCount := strings.Count(dataStr, "CHAT_MSG{")
	// 	fmt.Printf("📊 Blockchain Blocks: %d\n", blockCount)

	// 	// Show first and last few characters of blockchain data
	// 	if len(dataStr) > 100 {
	// 	// 	fmt.Printf("📄 Data Preview: %s...%s\n", dataStr[:50], dataStr[len(dataStr)-50:])
	// 	// } else {
	// 	// 	fmt.Printf("📄 Data: %s\n", dataStr)
	// 	// }
	// }

	// Log progress towards run completion (commented out for cleaner output)
	// fmt.Printf("📈 Progress: %d/12 blockchains received\n", messagesThisRun+1)
	// fmt.Printf("⏱️  Run Time: %s\n", time.Since(runStartTime).Round(time.Millisecond))
	// fmt.Println("🔗 END BLOCKCHAIN DATA 🔗")

	// Check for spam messages and log them
	if dataStr, ok := data.(string); ok {
		spamCount := strings.Count(dataStr, "SPAM_MSG{")
		if spamCount > 0 {
			fmt.Printf("🚨 SPAM DETECTED! Found %d spam messages in blockchain\n", spamCount)
			fmt.Printf("🚨 Spam content: %s\n", dataStr)
		}
	}

	// Store blockchain data for consensus analysis
	storeBlockchainForConsensus(blockchainMsg)

	// Process for consensus testing
	processMessageForConsensus("blockchain_message", "blockchain_node")
}

// storeBlockchainForConsensus stores blockchain data for consensus analysis
func storeBlockchainForConsensus(blockchainMsg map[string]interface{}) {
	// Extract node identifier from the blockchain message
	var nodeID string

	// Try to extract sender information
	if sender, exists := blockchainMsg["sender"]; exists {
		nodeID = fmt.Sprintf("%v", sender)
	} else if publicKey, exists := blockchainMsg["public_key"]; exists {
		// Use public key as fallback identifier
		pubKeyStr := fmt.Sprintf("%v", publicKey)
		// Store under the full public key for proper lookup
		nodeID = pubKeyStr
	} else {
		// Generate a unique identifier for this blockchain
		nodeID = fmt.Sprintf("node_%d", messagesThisRun)
	}

	// Store blockchain data for this node
	if nodeBlockchains[nodeID] == nil {
		nodeBlockchains[nodeID] = make([]string, 0)
	}

	// Convert blockchain data to string for storage
	blockchainData := fmt.Sprintf("%v", blockchainMsg["data"])
	nodeBlockchains[nodeID] = append(nodeBlockchains[nodeID], blockchainData)

	// fmt.Printf("💾 Stored blockchain for node %s (total: %d)\n", nodeID, len(nodeBlockchains[nodeID]))
}

// sendStartGossipingCommand sends a command to VM1 to start gossiping messages
func sendStartGossipingCommand() {
	// VM1's address (command listener on port 3001)
	vm1Address := "localhost:3001"

	fmt.Printf("📡 Sending start gossiping command to VM1 at %s...\n", vm1Address)

	// Connect to VM1
	conn, err := net.Dial("tcp", vm1Address)
	if err != nil {
		fmt.Printf("❌ Failed to connect to VM1: %v\n", err)
		return
	}
	defer conn.Close()

	// Create start gossiping command
	startCommand := map[string]interface{}{
		"type":      "start_gossiping",
		"run":       currentRun,
		"timestamp": time.Now().Unix(),
	}

	// Marshal and send command
	commandBytes, err := json.Marshal(startCommand)
	if err != nil {
		fmt.Printf("❌ Failed to marshal start command: %v\n", err)
		return
	}

	_, err = conn.Write(commandBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send start command to VM1: %v\n", err)
		return
	}

	// Read response from VM1
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("❌ Failed to read response from VM1: %v\n", err)
		return
	}

	response := string(buffer[:n])
	fmt.Printf("✅ VM1 response: %s\n", response)
}

// clearBlockchainsOnAllVMs sends clear blockchain commands to all VMs
func clearBlockchainsOnAllVMs() {
	fmt.Printf("🚨 SENDING CLEAR BLOCKCHAIN COMMAND for run %d\n", currentRun)
	fmt.Printf("⏰ Current time: %s\n", time.Now().Format("15:04:05.000"))

	// Only VM1 has a command listener, so we only send to VM1
	// VM1 will then gossip the clear command to other VMs
	vmAddresses := map[string]string{
		"VM1": "localhost:3001", // Only VM1's command listener
	}

	var wg sync.WaitGroup
	for vmName, address := range vmAddresses {
		wg.Add(1)
		go func(name, addr string) {
			defer wg.Done()
			sendClearBlockchainCommand(name, addr)
		}(vmName, address)
	}
	wg.Wait()
	fmt.Println("🧹 Blockchain clear commands sent to VM1 (will gossip to others)")
}

// sendClearBlockchainCommand sends a clear blockchain command to a specific VM
func sendClearBlockchainCommand(vmName, address string) {
	fmt.Printf("🧹 Sending clear blockchain command to %s at %s...\n", vmName, address)

	// Connect to VM
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("❌ Failed to connect to %s: %v\n", vmName, err)
		return
	}
	defer conn.Close()

	// Create clear blockchain command
	clearCommand := map[string]interface{}{
		"type":      "clear_blockchain",
		"run":       currentRun,
		"timestamp": time.Now().Unix(),
	}

	// Marshal and send command
	commandBytes, err := json.Marshal(clearCommand)
	if err != nil {
		fmt.Printf("❌ Failed to marshal clear command for %s: %v\n", vmName, err)
		return
	}

	_, err = conn.Write(commandBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send clear command to %s: %v\n", vmName, err)
		return
	}

	// Read response from VM
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("❌ Failed to read response from %s: %v\n", vmName, err)
		return
	}

	response := string(buffer[:n])
	fmt.Printf("✅ %s response: %s\n", vmName, response)
}

// handleStringMessage processes legacy string messages
func handleStringMessage(msg string) {
	fmt.Println("Received from peer:", msg)

	var filteredMessage string = msg

	if strings.HasPrefix("Message Reached To Peer ", msg) {
		filteredMessage = strings.Split(msg, "Message Reached To Peer ")[0]
	}

	switch filteredMessage {

	// Implementation
	case "Message Reached To Peer ":
		reachabilityCount++
		// Process for consensus testing (extract node ID from message)
		nodeID := extractNodeIDFromMessage(msg)
		processMessageForConsensus(msg, nodeID)
		break

	// Control
	case "Received Censored Message":
		reachabilityCount++
		str := strings.Split(filteredMessage, " ")[1]
		ts, err := strconv.ParseUint(str, 10, 64)

		if err != nil {
			fmt.Println("Error parsing uint " + err.Error())
		}

		timestamp_arrived = append(timestamp_arrived, ts)
		// Process for consensus testing
		processMessageForConsensus(msg, "censored_message_node")

		break
	}
}

func SendMessage(messageContent string, roomID string, port uint64) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	var wg sync.WaitGroup

	for _, peer := range peers {
		// ignore if peer in blockchain

		wg.Add(1)
		go func(peer peerDetails.Peer) {
			defer wg.Done()

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

		}(peer) // Pass peer as an argument to avoid closure capture issues
	}

	wg.Wait() // Wait for all goroutines to finish
	return nil
}

// displayFinalResults shows the overall consensus testing results
func displayFinalResults() {
	fmt.Println("\n=== FINAL CONSENSUS RESULTS ===")

	// Calculate average consensus integrity
	var avgIntegrity float64
	if len(consensusIntegrityResults) > 0 {
		total := 0.0
		for _, score := range consensusIntegrityResults {
			total += score
		}
		avgIntegrity = total / float64(len(consensusIntegrityResults))
	}

	// Calculate average attack success rate
	var avgASR float64
	if len(attackSuccessRates) > 0 {
		total := 0.0
		for _, asr := range attackSuccessRates {
			total += asr
		}
		avgASR = total / float64(len(attackSuccessRates))
	}

	// Calculate average latency
	var avgLatency int64
	if len(totalLatencies) > 0 {
		total := int64(0)
		for _, latency := range totalLatencies {
			total += latency
		}
		avgLatency = total / int64(len(totalLatencies))
	}

	fmt.Printf("Average Consensus Integrity: %.2f%%\n", avgIntegrity*100)
	fmt.Printf("Average Attack Success Rate: %.2f%%\n", avgASR*100)
	fmt.Printf("Average Latency: %d ms\n", avgLatency)

	// Display run-by-run breakdown
	fmt.Println("\n=== RUN BREAKDOWN ===")
	for i := 0; i < len(consensusIntegrityResults); i++ {
		fmt.Printf("Run %d: Integrity=%.1f%%, ASR=%.1f%%\n",
			i+1,
			consensusIntegrityResults[i]*100,
			attackSuccessRates[i]*100)
	}
}

// saveResultsToCSV saves all consensus testing results to a CSV file
func saveResultsToCSV() {
	filename := fmt.Sprintf("consensus_results_%s.csv", time.Now().Format("2006-01-02_15-04-05"))

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Run",
		"Consensus_Integrity_%",
		"Attack_Success_Rate_%",
		"Latency_ms",
		"Blockchains_Received",
		"Run_Status",
		"Timestamp",
	}
	err = writer.Write(header)
	if err != nil {
		fmt.Printf("Error writing CSV header: %v\n", err)
		return
	}

	// Write run-by-run data
	for i := 0; i < len(consensusIntegrityResults); i++ {
		var latency int64
		if i < len(totalLatencies) {
			latency = totalLatencies[i]
		}

		row := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%.2f", consensusIntegrityResults[i]*100),
			fmt.Sprintf("%.2f", attackSuccessRates[i]*100),
			fmt.Sprintf("%d", latency),
			"12", // Expected blockchains per run
			"COMPLETED",
			time.Now().Format("2006-01-02 15:04:05"),
		}

		err = writer.Write(row)
		if err != nil {
			fmt.Printf("Error writing CSV row %d: %v\n", i+1, err)
			continue
		}
	}

	// Write summary statistics
	writer.Write([]string{}) // Empty row for spacing
	writer.Write([]string{"SUMMARY_STATISTICS"})

	// Calculate averages
	var avgIntegrity float64
	var avgASR float64
	var avgLatency int64

	if len(consensusIntegrityResults) > 0 {
		total := 0.0
		for _, score := range consensusIntegrityResults {
			total += score
		}
		avgIntegrity = total / float64(len(consensusIntegrityResults))
	}

	if len(attackSuccessRates) > 0 {
		total := 0.0
		for _, asr := range attackSuccessRates {
			total += asr
		}
		avgASR = total / float64(len(attackSuccessRates))
	}

	if len(totalLatencies) > 0 {
		total := int64(0)
		for _, latency := range totalLatencies {
			total += latency
		}
		avgLatency = total / int64(len(totalLatencies))
	}

	// Write summary rows
	writer.Write([]string{"Total_Runs", fmt.Sprintf("%d", len(consensusIntegrityResults))})
	writer.Write([]string{"Average_Consensus_Integrity_%", fmt.Sprintf("%.2f", avgIntegrity*100)})
	writer.Write([]string{"Average_Attack_Success_Rate_%", fmt.Sprintf("%.2f", avgASR*100)})
	writer.Write([]string{"Average_Latency_ms", fmt.Sprintf("%d", avgLatency)})
	writer.Write([]string{"Test_Configuration", "4_nodes_2_honest_2_attacker"})
	writer.Write([]string{"Blockchains_Per_Run", "12"})
	writer.Write([]string{"Timeout_Per_Run", "60s"})

	fmt.Printf("Results saved to CSV file: %s\n", filename)
}
