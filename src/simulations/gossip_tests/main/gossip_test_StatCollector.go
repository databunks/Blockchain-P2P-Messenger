package main

import (
	"blockchain-p2p-messenger/src/peerDetails"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Gossip testing variables
var currentRun int = 0
var totalRuns int = 100
var messagesThisRun int = 0
var runStartTime time.Time
var messageInitiationTime time.Time   // Time when first message is sent
var lastMessageReceivedTime time.Time // Time when last message was received

// Gossip mode toggle
var gossipMode string = "control" // "control" for normal operation, "attack" for attack scenario
var expectedNodes int = 4         // Total number of nodes in the network
var expectedReachability int = 4  // Expected number of nodes to receive the message

// Gossip testing mutexes
var gossipMutex sync.Mutex
var isTestRunning bool = false
var attackerNodes []string // List of attacker node public keys
var honestNodes []string   // List of honest node public keys

// Enhanced tracking for gossip metrics
var nodeMessageReceipts map[string]bool // nodeID -> hasReceivedMessage for current run
var reachabilityResults []float64       // Store reachability percentages for each run
var attackSuccessRates []bool           // Store attack success/failure for each run
var totalLatencies []int64              // Store latencies for each run
var messageOverheadCounts []int         // Store message overhead for each run
var runCompletionTimes []time.Time      // Store completion time for each run
var nodeMessageReceiptsMutex sync.Mutex // Protect concurrent access to nodeMessageReceipts
var messagesMutex sync.Mutex            // Protect concurrent access to messagesThisRun
var testRunningMutex sync.Mutex         // Protect concurrent access to isTestRunning

// Message tracking
var totalMessagesSent int = 0
var messagesSentMutex sync.Mutex

func main() {
	// Initialize gossip testing
	initializeGossipTest()

	go ListenOnPort(":3002")

	fmt.Println("Gossip Test Stat Collector Started!")

	// Set gossip mode (change this to switch between modes)
	setGossipMode(gossipMode) // "attack" for attack scenario, "control" for normal operation

	fmt.Printf("Target: %d runs with %d nodes per run (Mode: %s)\n", totalRuns, expectedNodes, gossipMode)

	// Start the first run
	startNewRun()

	// Wait for all runs to complete
	waitForTestCompletion()

	fmt.Println("=== GOSSIP TEST COMPLETED ===")
	fmt.Printf("Total Runs: %d\n", currentRun)

	// Display final gossip results
	displayFinalResults()

	// Save results to CSV file
	saveResultsToCSV()
}

// initializeGossipTest sets up the gossip testing environment
func initializeGossipTest() {
	// Define attacker nodes (if any)
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
	nodeMessageReceiptsMutex.Lock()
	nodeMessageReceipts = make(map[string]bool)
	nodeMessageReceiptsMutex.Unlock()

	reachabilityResults = make([]float64, 0)
	attackSuccessRates = make([]bool, 0)
	totalLatencies = make([]int64, 0)
	messageOverheadCounts = make([]int, 0)
	runCompletionTimes = make([]time.Time, 0)

	fmt.Println("=== GOSSIP TEST INITIALIZATION ===")
	fmt.Printf("Total Nodes: %d\n", expectedNodes)
	fmt.Printf("Honest Nodes: %d\n", len(honestNodes))
	for i, key := range honestNodes {
		fmt.Printf("  Honest %d: %s\n", i+1, key[:16]+"...")
	}
	fmt.Printf("Attacker Nodes: %d\n", len(attackerNodes))
	for i, key := range attackerNodes {
		fmt.Printf("  Attacker %d: %s\n", i+1, key[:16]+"...")
	}
	fmt.Printf("Expected Reachability: %d nodes\n", expectedReachability)
	fmt.Println("VM1: Sends censored message")
	fmt.Println("VM2, VM3, VM4: Receive and forward messages")
	fmt.Println("All VMs: Send message receipts to stat collector")
}

// setGossipMode configures the test for different scenarios
func setGossipMode(mode string) {
	gossipMode = mode
	if mode == "attack" {
		expectedReachability = 3 // Expect only 3 nodes to receive message (attack scenario)
		fmt.Printf("🎯 Attack Mode: Expecting %d/%d nodes to receive message\n", expectedReachability, expectedNodes)
	} else {
		expectedReachability = 4 // Expect all 4 nodes to receive message (control scenario)
		fmt.Printf("🎯 Control Mode: Expecting %d/%d nodes to receive message\n", expectedReachability, expectedNodes)
	}
}

// startNewRun begins a new gossip test run
func startNewRun() {
	nodeMessageReceiptsMutex.Lock()
	defer nodeMessageReceiptsMutex.Unlock()

	if currentRun >= totalRuns {
		fmt.Println("All runs completed!")
		return
	}

	currentRun++
	messagesMutex.Lock()
	messagesThisRun = 0
	messagesMutex.Unlock()
	messageInitiationTime = time.Time{}   // Reset message initiation time for new run
	lastMessageReceivedTime = time.Time{} // Reset last message received time for new run
	testRunningMutex.Lock()
	isTestRunning = true
	testRunningMutex.Unlock()

	// Reset message tracking for new run
	nodeMessageReceipts = make(map[string]bool)

	// Reset message overhead counter
	messagesSentMutex.Lock()
	totalMessagesSent = 0
	messagesSentMutex.Unlock()

	// Verify complete run isolation
	fmt.Printf("🔍 Verifying complete run isolation...\n")
	if len(nodeMessageReceipts) > 0 {
		fmt.Printf("⚠️  WARNING: Old message receipts detected! Clearing again...\n")
		nodeMessageReceipts = make(map[string]bool)
	}
	fmt.Printf("✅ Run isolation verified - clean slate for run %d\n", currentRun)

	fmt.Printf("\n=== STARTING RUN %d/%d ===\n", currentRun, totalRuns)
	fmt.Printf("Expected: %d nodes to receive message (Mode: %s)\n", expectedReachability, gossipMode)
	fmt.Println("Sending start gossiping command to VM1...")

	// Set run start time RIGHT BEFORE sending command to VM1
	runStartTime = time.Now()
	fmt.Printf("Run started at: %s\n", runStartTime.Format("15:04:05"))

	// Send start gossiping command to VM1
	sendStartGossipingCommand()
}

// sendStartGossipingCommand sends the command to start the gossip test
func sendStartGossipingCommand() {
	fmt.Println("Sending start gossiping command to VM1...")

	// Set message initiation time when we send the command
	messageInitiationTime = time.Now()
	fmt.Printf("🎯 Message initiation started at: %s\n", messageInitiationTime.Format("15:04:05.000"))

	// Send command to VM1 on localhost:3001
	go sendStartCommandToNode("localhost", 3001, "SEND_LIMITED_MESSAGE:2")
}

// sendStartCommandToNode sends a start command to a specific node
func sendStartCommandToNode(nodeIP string, port int, command string) {
	address := net.JoinHostPort(nodeIP, fmt.Sprintf("%d", port))
	fmt.Printf("Sending command to %s: %s\n", address, command)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Error connecting to %s: %v\n", address, err)
		return
	}
	defer conn.Close()

	// Send the command
	_, err = conn.Write([]byte(command))
	if err != nil {
		fmt.Printf("Error sending command to %s: %v\n", address, err)
		return
	}

	fmt.Printf("✅ Start command sent to %s\n", address)
}

// isRunComplete checks if the current run is complete
func isRunComplete() bool {
	nodeMessageReceiptsMutex.Lock()
	defer nodeMessageReceiptsMutex.Unlock()

	// Run is complete when we've received messages from all expected nodes
	return len(nodeMessageReceipts) >= expectedReachability
}

// recordRunMetrics records metrics for a run, whether completed or timed out
func recordRunMetrics(completed bool) {
	// Calculate reachability based on current receipts
	reachability := calculateReachability()
	reachabilityResults = append(reachabilityResults, reachability)

	// Calculate latency (from message initiation to last message received)
	var latency int64
	if !messageInitiationTime.IsZero() && !lastMessageReceivedTime.IsZero() {
		latency = lastMessageReceivedTime.Sub(messageInitiationTime).Milliseconds()
	} else if !messageInitiationTime.IsZero() {
		// If no messages were received, use timeout time
		latency = time.Since(messageInitiationTime).Milliseconds()
	} else {
		// Fallback to run start time if message initiation time is not set
		latency = time.Since(runStartTime).Milliseconds()
	}
	totalLatencies = append(totalLatencies, latency)

	// Get message overhead for this run
	messagesSentMutex.Lock()
	overhead := totalMessagesSent
	messagesSentMutex.Unlock()
	messageOverheadCounts = append(messageOverheadCounts, overhead)

	// Determine attack success/failure
	attackSuccess := determineAttackSuccess(reachability)
	attackSuccessRates = append(attackSuccessRates, attackSuccess)

	// Store completion time for this run
	runCompletionTimes = append(runCompletionTimes, time.Now())

	// Display run results
	if completed {
		if attackSuccess {
			fmt.Printf("Run %d: Reachability: %.1f%% < 100%% → 🚨 ATTACK SUCCESSFUL (Latency: %d ms, Overhead: %d msgs)\n",
				currentRun, reachability*100, latency, overhead)
		} else {
			fmt.Printf("Run %d: Reachability: %.1f%% >= 100%% → ✅ ATTACK FAILED (Latency: %d ms, Overhead: %d msgs)\n",
				currentRun, reachability*100, latency, overhead)
		}
	} else {
		fmt.Printf("Run %d: TIMEOUT - Reachability: %.1f%% (Latency: %d ms, Overhead: %d msgs)\n",
			currentRun, reachability*100, latency, overhead)
	}
}

// waitForTestCompletion waits for all runs to finish
func waitForTestCompletion() {
	for currentRun < totalRuns {
		// Wait for current run to complete with timeout
		timeout := time.After(1 * time.Second) // 1 second timeout per run
		runCompleted := false
		for isTestRunning && !isRunComplete() {
			select {
			case <-timeout:
				fmt.Printf("⚠️  Run %d timed out after 1 second, moving to next run\n", currentRun)
				testRunningMutex.Lock()
				isTestRunning = false
				testRunningMutex.Unlock()
				runCompleted = false
				break
			default:
				time.Sleep(1 * time.Second)
			}
		}

		// Check if run completed successfully (not timed out)
		if !isTestRunning && isRunComplete() {
			runCompleted = true
		}

		// Record metrics for this run (whether completed or timed out)
		recordRunMetrics(runCompleted)

		if currentRun < totalRuns {
			// Start next run
			time.Sleep(3 * time.Second) // 3 second pause between runs
			startNewRun()
		}
	}
}

// processMessageForGossip handles messages for gossip testing
func processMessageForGossip(msg string, nodeID string) {
	gossipMutex.Lock()
	defer gossipMutex.Unlock()

	testRunningMutex.Lock()
	testRunning := isTestRunning
	testRunningMutex.Unlock()

	if !testRunning {
		return
	}

	// Increment message count for current run
	messagesMutex.Lock()
	messagesThisRun++
	messagesMutex.Unlock()

	// Record message receipt for this node
	nodeMessageReceiptsMutex.Lock()
	nodeMessageReceipts[nodeID] = true
	receiptCount := len(nodeMessageReceipts)
	lastMessageReceivedTime = time.Now() // Update last message received time
	nodeMessageReceiptsMutex.Unlock()

	// Safely truncate nodeID for display
	displayID := nodeID
	if len(nodeID) > 16 {
		displayID = nodeID[:16] + "..."
	}
	fmt.Printf("Run %d: Message receipt %d/%d from %s\n", currentRun, receiptCount, expectedReachability, displayID)

	// Check if run is complete
	if receiptCount >= expectedReachability {
		testRunningMutex.Lock()
		isTestRunning = false
		testRunningMutex.Unlock()

		fmt.Printf("Run %d: All %d messages received, run completed successfully\n", currentRun, expectedReachability)

		// Wait briefly to ensure all messages are properly sent before clearing
		fmt.Printf("⏳ Waiting 3 seconds to ensure all messages are sent...\n")
		time.Sleep(3 * time.Second)

		// Clear message tracking on all VMs before next run
		fmt.Printf("🧹 Clearing message tracking on all VMs for next run...\n")
		clearStartTime := time.Now()
		clearMessagesOnAllVMs()
		clearDuration := time.Since(clearStartTime).Milliseconds()
		fmt.Printf("⏱️  Message clearing completed in %d ms\n", clearDuration)

		// Wait briefly for cleanup to prevent cross-run contamination
		fmt.Printf("⏳ Waiting 3 seconds for message cleanup and isolation...\n")
		time.Sleep(3 * time.Second)

		// Brief barrier to ensure complete isolation
		fmt.Printf("🚧 Ensuring complete run isolation...\n")
		time.Sleep(3 * time.Second)
	}
}

// calculateReachability calculates the percentage of nodes that received the message
func calculateReachability() float64 {
	nodeMessageReceiptsMutex.Lock()
	defer nodeMessageReceiptsMutex.Unlock()

	if expectedReachability == 0 {
		return 0.0
	}

	// Count nodes that received the message
	receivedCount := len(nodeMessageReceipts)
	reachability := float64(receivedCount) / float64(expectedReachability)

	fmt.Printf("   📊 Reachability Calculation:\n")
	fmt.Printf("      - Nodes that received message: %d\n", receivedCount)
	fmt.Printf("      - Expected reachability: %d\n", expectedReachability)
	fmt.Printf("      - Reachability: %.1f%%\n", reachability*100)

	return reachability
}

// determineAttackSuccess determines if the attack was successful based on reachability
func determineAttackSuccess(reachability float64) bool {
	// Attack is successful if reachability falls below 100%
	// This means the censored message did not reach all intended recipients
	return reachability < 1.0
}

// clearMessagesOnAllVMs clears message tracking on all VMs
func clearMessagesOnAllVMs() {
	fmt.Println("Clearing message tracking on all VMs...")

	peers := peerDetails.GetPeersInRoom("room-xyz-987")
	for _, peer := range peers {
		fmt.Printf("Clearing messages on %s...\n", peer.IP)
		go sendClearCommandToNode(peer.IP, 3000, "CLEAR_MESSAGES")
	}

	// Wait for all clear commands to complete
	time.Sleep(1 * time.Second)
}

// sendClearCommandToNode sends a clear command to a specific node
func sendClearCommandToNode(nodeIP string, port int, command string) {
	address := net.JoinHostPort(nodeIP, fmt.Sprintf("%d", port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Error connecting to %s: %v\n", address, err)
		return
	}
	defer conn.Close()

	// Send the clear command
	_, err = conn.Write([]byte(command))
	if err != nil {
		fmt.Printf("Error sending clear command to %s: %v\n", address, err)
		return
	}

	fmt.Printf("✅ Clear command sent to %s\n", address)
}

// displayFinalResults shows the overall gossip testing results
func displayFinalResults() {
	fmt.Println("\n=== FINAL GOSSIP TEST RESULTS ===")
	fmt.Printf("Total Runs: %d\n", currentRun)
	fmt.Printf("Test Mode: %s\n", gossipMode)

	// Calculate average metrics
	var avgReachability float64
	var avgLatency float64
	var avgOverhead float64
	var attackSuccessCount int

	for i := 0; i < len(reachabilityResults); i++ {
		avgReachability += reachabilityResults[i]
		avgLatency += float64(totalLatencies[i])
		avgOverhead += float64(messageOverheadCounts[i])
		if attackSuccessRates[i] {
			attackSuccessCount++
		}
	}

	if len(reachabilityResults) > 0 {
		avgReachability /= float64(len(reachabilityResults))
		avgLatency /= float64(len(totalLatencies))
		avgOverhead /= float64(len(messageOverheadCounts))
	}

	attackSuccessRate := float64(attackSuccessCount) / float64(len(attackSuccessRates)) * 100

	fmt.Printf("\n📊 METRICS SUMMARY:\n")
	fmt.Printf("   - Average Reachability: %.1f%%\n", avgReachability*100)
	fmt.Printf("   - Average Latency: %.0f ms\n", avgLatency)
	fmt.Printf("   - Average Message Overhead: %.0f messages\n", avgOverhead)
	fmt.Printf("   - Attack Success Rate: %.1f%% (%d/%d runs)\n", attackSuccessRate, attackSuccessCount, len(attackSuccessRates))

	// Determine overall test result
	if avgReachability < 1.0 {
		fmt.Printf("\n🚨 OVERALL RESULT: DEFENSE FAILURE\n")
		fmt.Printf("   - Reachability below 100%% indicates message censorship\n")
		fmt.Printf("   - Attack was successful in %.1f%% of runs\n", attackSuccessRate)
	} else {
		fmt.Printf("\n✅ OVERALL RESULT: DEFENSE SUCCESS\n")
		fmt.Printf("   - 100%% reachability achieved\n")
		fmt.Printf("   - Message censorship prevented\n")
	}
}

// saveResultsToCSV saves all test results to a CSV file
func saveResultsToCSV() {
	filename := fmt.Sprintf("gossip_test_results_%s_%s.csv", gossipMode, time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Run", "Reachability(%)", "Latency(ms)", "MessageOverhead", "AttackSuccess", "CompletionTime"}
	writer.Write(header)

	// Write data
	for i := 0; i < len(reachabilityResults); i++ {
		row := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%.1f", reachabilityResults[i]*100),
			fmt.Sprintf("%d", totalLatencies[i]),
			fmt.Sprintf("%d", messageOverheadCounts[i]),
			fmt.Sprintf("%t", attackSuccessRates[i]),
			runCompletionTimes[i].Format("15:04:05.000"),
		}
		writer.Write(row)
	}

	// Calculate and write summary row with attack success rate
	var attackSuccessCount int
	for _, success := range attackSuccessRates {
		if success {
			attackSuccessCount++
		}
	}
	attackSuccessRate := float64(attackSuccessCount) / float64(len(attackSuccessRates)) * 100

	// Write summary row
	summaryRow := []string{
		"SUMMARY",
		fmt.Sprintf("%.1f", attackSuccessRate),
		fmt.Sprintf("%d", attackSuccessCount),
		fmt.Sprintf("%d", len(attackSuccessRates)),
		"Attack Success Rate (%)",
		time.Now().Format("15:04:05.000"),
	}
	writer.Write(summaryRow)

	fmt.Printf("✅ Results saved to %s\n", filename)
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

		// Read message sent by client
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			return
		}

		// Unmarshal message (you JSON-encoded it before sending)
		var msg string
		err = json.Unmarshal(buffer[:n], &msg)
		if err != nil {
			fmt.Println("Error unmarshaling message:", err)
			return
		}

		// Process the message
		processGossipMessage(msg)

		// Response back to client
		_, err = conn.Write([]byte("Message received!\n"))
		if err != nil {
			fmt.Printf("Error writing to client: %v\n", err)
		}
	}
}

// processGossipMessage processes incoming gossip messages
func processGossipMessage(msg string) {
	fmt.Println("Received from peer:", msg)

	// Extract node ID from message (you may need to modify this based on your message format)
	nodeID := extractNodeIDFromMessage(msg)

	// Process different message types
	if strings.Contains(msg, "Message Reached To Peer") {
		// Message receipt confirmation
		processMessageForGossip(msg, nodeID)
	} else if strings.Contains(msg, "Received Censored Message") {
		// Censored message receipt
		processMessageForGossip(msg, nodeID)
	} else if strings.Contains(msg, "Message Sent") {
		// Message overhead tracking
		messagesSentMutex.Lock()
		totalMessagesSent++
		messagesSentMutex.Unlock()
	} else if strings.Contains(msg, "GOSSIP_START") {
		// Gossip test started
		messageInitiationTime = time.Now()
		fmt.Printf("🎯 Gossip test initiated at: %s\n", messageInitiationTime.Format("15:04:05.000"))
	}
}

// extractNodeIDFromMessage extracts the node ID from a message
func extractNodeIDFromMessage(msg string) string {
	// This is a placeholder - you'll need to implement based on your message format
	// For now, return a simple identifier
	return fmt.Sprintf("node_%d", time.Now().UnixNano()%1000)
}

func SendMessage(messageContent string, roomID string, port uint64) error {
	peers := peerDetails.GetPeersInRoom(roomID)

	var wg sync.WaitGroup

	for _, peer := range peers {
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
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				log.Printf("Error reading response from %s: %v\n", peer.IP, err)
				return
			}

			// Print the response received from the peer
			response := string(buffer[:n])
			fmt.Printf("Response from %s: %s\n", peer.IP, response)

		}(peer)
	}

	wg.Wait()
	return nil
}
