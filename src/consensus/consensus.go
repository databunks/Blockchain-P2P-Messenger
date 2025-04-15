package consensus

import (
	"blockchain-p2p-messenger/src/network"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/anthdm/hbbft"
	"github.com/hashicorp/raft"
)

type Transaction struct {
	Data string
}

func (t Transaction) Hash() []byte {
	return []byte(t.Data)
}

var (
	HB       *hbbft.HoneyBadger
	raftNode *raft.Raft
	net      *network.Network
)

type NetworkTransport struct {
	net *network.Network
}

func (t *NetworkTransport) Send(to uint64, msg []byte) error {
	fmt.Printf("Sending message to node %d: %v\n", to, msg)
	return t.net.Send(fmt.Sprintf("%d", to), msg)
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

// getYggdrasilPeers returns a list of Yggdrasil peer IP addresses
func getYggdrasilPeers() ([]string, error) {
	cmd := exec.Command("sudo", "yggdrasilctl", "getPeers")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse the output to get peer IP addresses
	var peers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Skip header line and empty lines
		if strings.Contains(line, "IP Address") || strings.TrimSpace(line) == "" {
			continue
		}

		// Extract IP address (4th field)
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			peers = append(peers, fields[3])
		}
	}
	return peers, nil
}

func InitConsensus(publicKey string, port int) error {
	// Get Yggdrasil peers
	peers, err := getYggdrasilPeers()
	if err != nil {
		return fmt.Errorf("failed to get Yggdrasil peers: %v", err)
	}

	fmt.Printf("Initializing consensus with Yggdrasil peers: %v\n", peers)

	// Initialize network
	net = network.NewNetwork(publicKey)

	// Add Yggdrasil peers
	for i, peerIP := range peers {
		net.AddPeer(fmt.Sprintf("%d", i+1), peerIP)
	}

	if err := net.Start(port); err != nil {
		return err
	}

	transport := &NetworkTransport{net: net}

	// Use Raft for small networks (2-3 nodes)
	if len(peers) < 4 {
		fmt.Println("Using Raft consensus for small network")
		// Initialize Raft configuration
		config := raft.DefaultConfig()
		config.LocalID = raft.ServerID("1") // Use "1" as our ID

		// Create Raft transport
		raftTransport := &RaftTransport{transport: transport}

		// Initialize Raft
		var err error
		logStore := raft.NewInmemStore()
		stableStore := raft.NewInmemStore()
		snapshotStore := raft.NewInmemSnapshotStore()
		raftNode, err = raft.NewRaft(config, &FSM{}, logStore, stableStore, snapshotStore, raftTransport)
		if err != nil {
			return err
		}

		// Bootstrap the cluster
		if len(peers) == 1 {
			configuration := raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      config.LocalID,
						Address: raft.ServerAddress(peers[0]),
					},
				},
			}
			raftNode.BootstrapCluster(configuration)
		}
	} else {
		// Use Honey Badger BFT for larger networks
		fmt.Println("Using Honey Badger BFT consensus")

		// Convert IP addresses to uint64 IDs
		nodeIDs := make([]uint64, len(peers))
		for i := range peers {
			nodeIDs[i] = uint64(i + 1)
		}

		cfg := hbbft.Config{
			N:         len(peers),
			F:         (len(peers) - 1) / 3,
			ID:        uint64(1), // Use 1 as our ID
			Nodes:     nodeIDs,
			BatchSize: 100,
		}

		HB = hbbft.NewHoneyBadger(cfg)

		// Handle incoming messages
		go func() {
			fmt.Printf("Node %d: Starting message handler\n", cfg.ID)
			for msg := range transport.Recv() {
				fmt.Printf("Node %d: Processing message: %v\n", cfg.ID, msg)
				if err := HB.HandleMessage(0, 0, &hbbft.ACSMessage{Payload: msg}); err != nil {
					fmt.Printf("Error handling message: %v\n", err)
				}
			}
		}()
	}

	return nil
}

func AddTransaction(data string) {
	fmt.Printf("Adding transaction: %s\n", data)
	if HB != nil {
		// Use Honey Badger BFT
		tx := Transaction{Data: data}
		HB.AddTransaction(tx)
	} else if raftNode != nil {
		// Use Raft
		raftNode.Apply([]byte(data), time.Second)
	}
}

func StartConsensus() {
	fmt.Println("Starting consensus engine")
	if HB != nil {
		// Start Honey Badger BFT in a goroutine
		go func() {
			HB.Start()
		}()
	}
	// Raft starts automatically
}

func GetOutputs() map[uint64][]Transaction {
	fmt.Println("Getting outputs")
	outputs := make(map[uint64][]Transaction)

	if HB != nil {
		// Get Honey Badger BFT outputs
		for epoch, txs := range HB.Outputs() {
			var transactions []Transaction
			for _, tx := range txs {
				if t, ok := tx.(Transaction); ok {
					transactions = append(transactions, t)
				}
			}
			outputs[epoch] = transactions
		}
	} else if raftNode != nil {
		// Get Raft outputs (simplified for demo)
		outputs[0] = []Transaction{{Data: "Raft consensus output"}}
	}

	return outputs
}

// RaftTransport implements raft.Transport
type RaftTransport struct {
	transport *NetworkTransport
}

func (t *RaftTransport) AppendEntriesPipeline(id raft.ServerID, target raft.ServerAddress) (raft.AppendPipeline, error) {
	return nil, nil
}

func (t *RaftTransport) DecodePeer(peer []byte) raft.ServerAddress {
	return raft.ServerAddress(string(peer))
}

func (t *RaftTransport) EncodePeer(id raft.ServerID, peer raft.ServerAddress) []byte {
	return []byte(peer)
}

func (t *RaftTransport) InstallSnapshot(id raft.ServerID, target raft.ServerAddress, args *raft.InstallSnapshotRequest, resp *raft.InstallSnapshotResponse, data io.Reader) error {
	return nil
}

func (t *RaftTransport) SetHeartbeatHandler(cb func(rpc raft.RPC)) {
}

func (t *RaftTransport) TimeoutNow(id raft.ServerID, target raft.ServerAddress, args *raft.TimeoutNowRequest, resp *raft.TimeoutNowResponse) error {
	return nil
}

func (t *RaftTransport) Consumer() <-chan raft.RPC {
	// Implement RPC consumer
	return make(chan raft.RPC)
}

func (t *RaftTransport) LocalAddr() raft.ServerAddress {
	return raft.ServerAddress("localhost")
}

func (t *RaftTransport) AppendEntries(id raft.ServerID, target raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
	// Implement append entries
	return nil
}

func (t *RaftTransport) RequestVote(id raft.ServerID, target raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	// Implement request vote
	return nil
}

// FSM implements raft.FSM
type FSM struct{}

func (f *FSM) Apply(log *raft.Log) interface{} {
	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (f *FSM) Restore(io.ReadCloser) error {
	return nil
}
