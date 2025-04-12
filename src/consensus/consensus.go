package consensus

import (
	"blockchain-p2p-messenger/src/network"
	"fmt"

	"github.com/anthdm/hbbft"
)

type Transaction struct {
	Data string
}

func (t Transaction) Hash() []byte {
	return []byte(t.Data)
}

var (
	HB  *hbbft.HoneyBadger
	net *network.Network
)

type NetworkTransport struct {
	net *network.Network
}

func (t *NetworkTransport) Send(to uint64, msg []byte) error {
	fmt.Printf("Sending message to node %d: %v\n", to, msg)
	return t.net.Send(to, msg)
}

func (t *NetworkTransport) Recv() <-chan []byte {
	ch := make(chan []byte, 100)
	go func() {
		fmt.Println("Starting message receiver")
		for msg := range t.net.Receive() {
			fmt.Printf("Received message from node %d: %v\n", msg.From, msg.Payload)
			ch <- msg.Payload
		}
	}()
	return ch
}

func InitConsensus(nodeID uint64, nodeIDs []uint64, port int) error {
	fmt.Printf("Initializing consensus for node %d with peers %v\n", nodeID, nodeIDs)

	// Initialize network
	net = network.NewNetwork(nodeID)

	// Add peers with correct ports
	for _, id := range nodeIDs {
		peerPort := port + int(id) - 1 // Each node gets its own port
		if id == nodeID {
			// Add ourselves with the correct port
			net.AddPeer(id, fmt.Sprintf("localhost:%d", peerPort))
		} else {
			// Add other peers with their respective ports
			net.AddPeer(id, fmt.Sprintf("localhost:%d", peerPort))
		}
	}

	if err := net.Start(port + int(nodeID) - 1); err != nil {
		return err
	}

	transport := &NetworkTransport{net: net}

	cfg := hbbft.Config{
		N:         len(nodeIDs),
		F:         (len(nodeIDs) - 1) / 3, // Maximum number of faulty nodes
		ID:        nodeID,
		Nodes:     nodeIDs,
		BatchSize: 100,
	}

	fmt.Printf("Creating HoneyBadger instance with config: N=%d, F=%d, ID=%d\n", cfg.N, cfg.F, cfg.ID)
	HB = hbbft.NewHoneyBadger(cfg)

	// Handle incoming messages
	go func() {
		fmt.Printf("Node %d: Starting message handler\n", nodeID)
		for msg := range transport.Recv() {
			fmt.Printf("Node %d: Processing message: %v\n", nodeID, msg)
			// Handle the message using HB's HandleMessage method
			if err := HB.HandleMessage(0, 0, &hbbft.ACSMessage{Payload: msg}); err != nil {
				fmt.Printf("Error handling message: %v\n", err)
			}
		}
	}()

	return nil
}

func AddTransaction(data string) {
	fmt.Printf("Adding transaction: %s\n", data)
	tx := Transaction{Data: data}
	HB.AddTransaction(tx)
}

func StartConsensus() {
	fmt.Println("Starting consensus engine")
	// Start the consensus engine in a goroutine to prevent blocking
	go func() {
		HB.Start()
	}()
}

func GetOutputs() map[uint64][]Transaction {
	fmt.Println("Getting outputs")
	outputs := make(map[uint64][]Transaction)
	for epoch, txs := range HB.Outputs() {
		var transactions []Transaction
		for _, tx := range txs {
			if t, ok := tx.(Transaction); ok {
				transactions = append(transactions, t)
			}
		}
		outputs[epoch] = transactions
	}
	return outputs
}
