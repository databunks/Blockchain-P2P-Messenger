package digitalsignatureforgerytests

import (
	"blockchain-p2p-messenger/src/network"
	"blockchain-p2p-messenger/src/peerDetails"
	"crypto/ed25519"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func DigitalSignatureForgeryTest(roomID string, port uint64, attackMode bool) bool {
	// Construct a custom random private / public key pair (Recieved from running src/genkeys/genRandKeys)
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	messageContent := "Digital Signature Forgery Test"

	// Construct our own message to send
	message := network.Message{
		PublicKey: "0000005ed266dc58d687b6ed84af4b4657162033cf379e9d8299bba941ae66e0", // stealing public key of a group member
		Message:   messageContent,
		RoomID:    roomID,
		Type:      "chat",
		Timestamp: uint64(time.Now().Unix()),
	}
	// Sign the message with our random forged private / public key pair

	if attackMode {
		// Sign a message using the private key
		// message := []byte("Hello, this is a test message.")
		signature := ed25519.Sign(priv, []byte(message.Message))
		message.DigitalSignature = hex.EncodeToString(signature)
	} else {
		signatureBytes := network.SignMessage([]byte(message.Message))
		message.DigitalSignature = hex.EncodeToString(signatureBytes)
	}

	// message.DigitalSignature = network.SignMessage([]byte(message.Message))

	peers := peerDetails.GetPeersInRoom(roomID)

	// Only assuming 1 peer for this demonstration
	peer := peers[0]

	// Dial peer
	fmt.Printf("Establishing connection with %s, %s.......\n", peer.IP, peer.PublicKey)
	address := net.JoinHostPort(peer.IP, fmt.Sprintf("%d", port))
	fmt.Println(address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Error connecting to %s: %v\n", peer.IP, err)
		return false
	}
	defer conn.Close()

	// Marshal message
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message for %s: %v\n", peer.IP, err)
		return false
	}

	// Send message
	_, err = conn.Write(msgBytes)
	if err != nil {
		log.Printf("Error sending message to %s: %v\n", peer.IP, err)
		return false
	}

	// Read the response from the peer
	buffer := make([]byte, 1024) // Buffer to store incoming data
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading response from %s: %v\n", peer.IP, err)
		return false
	}

	// Print the response received from the peer
	response := string(buffer[:n])

	fmt.Printf("Response from %s: %s\n", peer.IP, response)

	fmt.Println(response)

	if strings.Contains(response, "Invalid Digital Signature!") {
		return false
	} else if strings.Contains(response, "Authenticated") {
		return true
	}

	return false
}

// RunDigitalSignatureForgeryTestScenarios runs the digital signature forgery test under attack and control modes
func RunDigitalSignatureForgeryTestScenarios(roomID string, port uint64, totalTests int) {
	fmt.Printf("=== DIGITAL SIGNATURE FORGERY TEST SCENARIOS ===\n")
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("Room ID: %s\n", roomID)
	fmt.Printf("Port: %d\n\n", port)

	// Test metrics
	var attackModeResults []TestResult
	var controlModeResults []TestResult

	// Run Attack Mode Tests
	fmt.Println("=== ATTACK MODE TESTS ===")
	for i := 0; i < totalTests; i++ {
		fmt.Printf("Attack Test %d/%d\n", i+1, totalTests)

		startTime := time.Now()
		success := DigitalSignatureForgeryTest(roomID, port, true) // true = attack mode
		latency := time.Since(startTime)

		result := TestResult{
			TestNumber: i + 1,
			Mode:       "Attack",
			Success:    success,
			Latency:    latency,
		}

		attackModeResults = append(attackModeResults, result)

		// Brief pause between tests
		time.Sleep(100 * time.Millisecond)
	}

	// Run Control Mode Tests
	fmt.Println("\n=== CONTROL MODE TESTS ===")
	for i := 0; i < totalTests; i++ {
		fmt.Printf("Control Test %d/%d\n", i+1, totalTests)

		startTime := time.Now()
		success := DigitalSignatureForgeryTest(roomID, port, false) // false = control mode
		latency := time.Since(startTime)

		result := TestResult{
			TestNumber: i + 1,
			Mode:       "Control",
			Success:    success,
			Latency:    latency,
		}

		controlModeResults = append(controlModeResults, result)

		// Brief pause between tests
		time.Sleep(100 * time.Millisecond)
	}

	// Save results to CSV file
	saveResultsToCSV(attackModeResults, controlModeResults, roomID)

	// Evaluate and display results
	evaluateAndDisplayResults(attackModeResults, controlModeResults)
}

// TestResult represents the result of a single forgery test
type TestResult struct {
	TestNumber   int
	Mode         string
	Success      bool
	Response     string
	Latency      time.Duration
	PeerIP       string
	ErrorMessage string
}

// runSingleForgeryTest runs a single forgery test and returns the result
func runSingleForgeryTest(roomID string, port uint64, isAttackMode bool) TestResult {
	result := TestResult{}

	// Generate forged keys for attack mode, legitimate keys for control mode
	var pub ed25519.PublicKey
	var priv ed25519.PrivateKey
	var err error

	if isAttackMode {
		// Attack mode: Generate random forged keys
		pub, priv, err = ed25519.GenerateKey(nil)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to generate forged keys: %v", err)
			return result
		}
	} else {
		// Control mode: Use legitimate keys (you might want to load these from keydetails.env)
		// For now, we'll use a different message content to simulate legitimate vs forged
		pub, priv, err = ed25519.GenerateKey(nil)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to generate legitimate keys: %v", err)
			return result
		}
	}

	messageContent := "Digital Signature Forgery Test"
	if !isAttackMode {
		messageContent = "Legitimate Control Message"
	}

	// Construct message
	message := network.Message{
		PublicKey: hex.EncodeToString(pub),
		Message:   messageContent,
		RoomID:    roomID,
		Type:      "chat",
		Timestamp: uint64(time.Now().Unix()),
	}

	// Sign the message
	signature := ed25519.Sign(priv, []byte(message.Message))
	message.DigitalSignature = hex.EncodeToString(signature)

	peers := peerDetails.GetPeersInRoom(roomID)
	if len(peers) == 0 {
		result.ErrorMessage = "No peers found in room"
		return result
	}

	// Test with first peer only for consistent results
	peer := peers[0]
	result.PeerIP = peer.IP

	// Establish connection
	address := net.JoinHostPort(peer.IP, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer conn.Close()

	// Send message
	msgBytes, err := json.Marshal(message)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Marshaling failed: %v", err)
		return result
	}

	_, err = conn.Write(msgBytes)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Send failed: %v", err)
		return result
	}

	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // 5 second timeout
	n, err := conn.Read(buffer)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Read failed: %v", err)
		return result
	}

	response := string(buffer[:n])
	result.Response = response

	// Determine success based on response
	if isAttackMode {
		// Attack mode: Success if we get "Authenticated" (forgery succeeded)
		result.Success = strings.Contains(response, "Authenticated")
	} else {
		// Control mode: Success if we get "Authenticated" (legitimate message succeeded)
		result.Success = strings.Contains(response, "Authenticated")
	}

	return result
}

// evaluateAndDisplayResults evaluates the test results and displays metrics
func evaluateAndDisplayResults(attackResults, controlResults []TestResult) {
	fmt.Println("\n=== EVALUATION RESULTS ===")

	// Attack Mode Metrics
	fmt.Println("\n--- ATTACK MODE METRICS ---")
	attackMetrics := calculateMetrics(attackResults)
	displayMetrics("Attack", attackMetrics)

	// Control Mode Metrics
	fmt.Println("\n--- CONTROL MODE METRICS ---")
	controlMetrics := calculateMetrics(controlResults)
	displayMetrics("Control", controlMetrics)

	// Overall Assessment
	fmt.Println("\n--- OVERALL ASSESSMENT ---")
	if attackMetrics.SuccessRate < 0.01 { // 1% threshold
		fmt.Println("✅ ATTACK FAILED: Success rate below 1% threshold")
	} else {
		fmt.Println("🚨 ATTACK SUCCESSFUL: Success rate above 1% threshold")
	}

	fmt.Printf("Attack Success Rate: %.2f%%\n", attackMetrics.SuccessRate*100)
	fmt.Printf("Control Success Rate: %.2f%%\n", controlMetrics.SuccessRate*100)
}

// Metrics represents the calculated metrics for a test mode
type Metrics struct {
	TotalTests      int
	SuccessfulTests int
	SuccessRate     float64
	AvgLatency      time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
}

// calculateMetrics calculates metrics from test results
func calculateMetrics(results []TestResult) Metrics {
	if len(results) == 0 {
		return Metrics{}
	}

	var totalLatency time.Duration
	successfulTests := 0
	minLatency := results[0].Latency
	maxLatency := results[0].Latency

	for _, result := range results {
		if result.Success {
			successfulTests++
		}
		totalLatency += result.Latency

		if result.Latency < minLatency {
			minLatency = result.Latency
		}
		if result.Latency > maxLatency {
			maxLatency = result.Latency
		}
	}

	avgLatency := totalLatency / time.Duration(len(results))
	successRate := float64(successfulTests) / float64(len(results))

	return Metrics{
		TotalTests:      len(results),
		SuccessfulTests: successfulTests,
		SuccessRate:     successRate,
		AvgLatency:      avgLatency,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
	}
}

// displayMetrics displays metrics for a specific test mode
func displayMetrics(mode string, metrics Metrics) {
	fmt.Printf("Total Tests: %d\n", metrics.TotalTests)
	fmt.Printf("Successful Tests: %d\n", metrics.SuccessfulTests)
	fmt.Printf("Success Rate: %.2f%%\n", metrics.SuccessRate*100)
	fmt.Printf("Average Latency: %v\n", metrics.AvgLatency)
	fmt.Printf("Min Latency: %v\n", metrics.MinLatency)
	fmt.Printf("Max Latency: %v\n", metrics.MaxLatency)
}

// saveResultsToCSV saves all test results to a CSV file
func saveResultsToCSV(attackResults, controlResults []TestResult, roomID string) {
	// Create filename with timestamp
	filename := fmt.Sprintf("digital_signature_forgery_results_%s_%s.csv", roomID, time.Now().Format("20060102_150405"))

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
		"Test_Number",
		"Mode",
		"Success",
		"Latency_ms",
		"Response",
		"Peer_IP",
		"ErrorMessage",
		"Timestamp",
	}
	err = writer.Write(header)
	if err != nil {
		fmt.Printf("Error writing CSV header: %v\n", err)
		return
	}

	// Write attack mode results
	for _, result := range attackResults {
		row := []string{
			fmt.Sprintf("%d", result.TestNumber),
			result.Mode,
			fmt.Sprintf("%t", result.Success),
			fmt.Sprintf("%d", result.Latency.Milliseconds()),
			result.Response,
			result.PeerIP,
			result.ErrorMessage,
			time.Now().Format("2006-01-02 15:04:05"),
		}
		err = writer.Write(row)
		if err != nil {
			fmt.Printf("Error writing attack result row: %v\n", err)
		}
	}

	// Write control mode results
	for _, result := range controlResults {
		row := []string{
			fmt.Sprintf("%d", result.TestNumber),
			result.Mode,
			fmt.Sprintf("%t", result.Success),
			fmt.Sprintf("%d", result.Latency.Milliseconds()),
			result.Response,
			result.PeerIP,
			result.ErrorMessage,
			time.Now().Format("2006-01-02 15:04:05"),
		}
		err = writer.Write(row)
		if err != nil {
			fmt.Printf("Error writing control result row: %v\n", err)
		}
	}

	// Write summary statistics
	writer.Write([]string{}) // Empty row for spacing
	writer.Write([]string{"SUMMARY_STATISTICS"})

	// Calculate and write attack mode summary
	attackMetrics := calculateMetrics(attackResults)
	writer.Write([]string{"Attack_Mode_Total_Tests", fmt.Sprintf("%d", attackMetrics.TotalTests)})
	writer.Write([]string{"Attack_Mode_Successful_Tests", fmt.Sprintf("%d", attackMetrics.SuccessfulTests)})
	writer.Write([]string{"Attack_Mode_Success_Rate_%", fmt.Sprintf("%.2f", attackMetrics.SuccessRate*100)})
	writer.Write([]string{"Attack_Mode_Avg_Latency_ms", fmt.Sprintf("%d", attackMetrics.AvgLatency.Milliseconds())})
	writer.Write([]string{"Attack_Mode_Min_Latency_ms", fmt.Sprintf("%d", attackMetrics.MinLatency.Milliseconds())})
	writer.Write([]string{"Attack_Mode_Max_Latency_ms", fmt.Sprintf("%d", attackMetrics.MaxLatency.Milliseconds())})

	// Calculate and write control mode summary
	controlMetrics := calculateMetrics(controlResults)
	writer.Write([]string{"Control_Mode_Total_Tests", fmt.Sprintf("%d", controlMetrics.TotalTests)})
	writer.Write([]string{"Control_Mode_Successful_Tests", fmt.Sprintf("%d", controlMetrics.SuccessfulTests)})
	writer.Write([]string{"Control_Mode_Success_Rate_%", fmt.Sprintf("%.2f", controlMetrics.SuccessRate*100)})
	writer.Write([]string{"Control_Mode_Avg_Latency_ms", fmt.Sprintf("%d", controlMetrics.AvgLatency.Milliseconds())})
	writer.Write([]string{"Control_Mode_Min_Latency_ms", fmt.Sprintf("%d", controlMetrics.MinLatency.Milliseconds())})
	writer.Write([]string{"Control_Mode_Max_Latency_ms", fmt.Sprintf("%d", controlMetrics.MaxLatency.Milliseconds())})

	// Overall assessment
	writer.Write([]string{"Overall_Assessment"})
	if attackMetrics.SuccessRate < 0.01 {
		writer.Write([]string{"Attack_Result", "FAILED (Below 1% threshold)"})
	} else {
		writer.Write([]string{"Attack_Result", "SUCCESSFUL (Above 1% threshold)"})
	}
	writer.Write([]string{"Test_Configuration", fmt.Sprintf("Room: %s, Total Tests: %d", roomID, len(attackResults))})
	writer.Write([]string{"Failure_Threshold", "1%"})
	writer.Write([]string{"Test_Date", time.Now().Format("2006-01-02 15:04:05")})

	fmt.Printf("Results saved to CSV file: %s\n", filename)
}

// RunDigitalSignatureForgeryTest500 runs the test with 500 runs by default
func RunDigitalSignatureForgeryTest500(roomID string, port uint64) {
	fmt.Println("Running Digital Signature Forgery Test with 500 runs...")
	RunDigitalSignatureForgeryTestScenarios(roomID, port, 500)
}
