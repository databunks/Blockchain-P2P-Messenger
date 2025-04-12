package main

import (
	//"blockchain-p2p-messenger/src/genkeys"
	"blockchain-p2p-messenger/src/consensus"
	"blockchain-p2p-messenger/src/peerDetails"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Parse command line arguments
	nodeID := flag.Uint64("id", 1, "Node ID")
	port := flag.Int("port", 8001, "Port number")
	peers := flag.String("peers", "1,2,3,4", "Comma-separated list of node IDs")
	flag.Parse()

	// Parse peer IDs
	peerIDs := make([]uint64, 0)
	for _, idStr := range strings.Split(*peers, ",") {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid peer ID: %s", idStr))
		}
		peerIDs = append(peerIDs, id)
	}

	// Initialize peer details (for demo purposes)
	peerDetails.AddPeer("0000001b34f2c2b897d0354eee6f9c898082fa4a42792b8e45768448fb8eb62a", "21b:4cb0:d3d4:7682:fcab:1119:637:67f7", true)

	// Initialize consensus
	if err := consensus.InitConsensus(*nodeID, peerIDs, *port); err != nil {
		panic(err)
	}

	fmt.Println("Starting consensus")

	// Start the consensus engine
	consensus.StartConsensus()

	fmt.Println("Consensus started")

	// Add a transaction
	consensus.AddTransaction(fmt.Sprintf("Hello from node %d", *nodeID))

	// Wait for consensus
	time.Sleep(5 * time.Second)

	// Get and print outputs
	outputs := consensus.GetOutputs()
	for epoch, txs := range outputs {
		for _, tx := range txs {
			fmt.Printf("Node %d - Epoch: %d, Data: %s\n", *nodeID, epoch, tx.Data)
		}
	}

	// Keep the program running
	select {}
}
