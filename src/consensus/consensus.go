package consensus

import (

	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"log"
	"sync"
	"time"
	"github.com/anthdm/hbbft"
)

const (
	batchSize = 500
	numCores  = 4
)

type Message struct {
	From    uint64
	Payload hbbft.MessageTuple
}

var (
	txDelay  = (3 * time.Millisecond) / numCores
	messages = make(chan Message, 1024*1024)
	relayCh  = make(chan *Transaction, 1024*1024)
)

func (t Transaction) Hash() []byte {
	return []byte(t.digitalSignature)
}


type Transaction struct {
	senderPublicKey string
	typeofmessage string
	digitalSignature string
	timestamp uint64
}

func NewTransaction(senderPublicKey string, typeofmessage string, digitalSignature string, timestamp uint64) *Transaction {
	return &Transaction{senderPublicKey: senderPublicKey, typeofmessage: typeofmessage, digitalSignature: digitalSignature, timestamp: timestamp}
}


type Node struct {
	ID          uint64
	HB          *hbbft.HoneyBadger
	Transport   hbbft.Transport
	RpcCh       <-chan hbbft.RPC
	Lock        sync.RWMutex
	Mempool     map[string]*Transaction
	TotalCommit int
	Start       time.Time
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




// func NewTransaction() *Transaction {
// 	return &Transaction{rand.Uint64()}
// }

func NewNode(id uint64, tr hbbft.Transport, nodes []uint64) *Node {
	hb := hbbft.NewHoneyBadger(hbbft.Config{
		N:         len(nodes),
		ID:        id,
		Nodes:     nodes,
		BatchSize: batchSize,
	})
	return &Node{
		ID:        id,
		Transport: tr,
		HB:        hb,
		RpcCh:     tr.Consume(),
		Mempool:   make(map[string]*Transaction),
		Start:     time.Now(),
	}
}

func MakeNetwork(n int, nodeIDs []uint64) []*Node {
	transports := make([]hbbft.Transport, n)
	nodes := make([]*Node, n)
	for i := 0; i < n; i++ {
		transports[i] = hbbft.NewLocalTransport(uint64(i))
		nodes[i] = NewNode(uint64(i), transports[i], nodeIDs)
	}
	connectTransports(transports)
	return nodes
}

func connectTransports(tt []hbbft.Transport) {
	for i := 0; i < len(tt); i++ {
		for ii := 0; ii < len(tt); ii++ {
			if ii == i {
				continue
			}
			tt[i].Connect(tt[ii].Addr(), tt[ii])
		}
	}
}

// func (n *Node) ReceiveTransaction(strtx string){

// 	// tx will just be a message for now but later we add more
// 	n.HB.AddTransaction(NewTransaction(strtx))

// }

// Hash implements the hbbft.Transaction interface.
// func (t *Transaction) Hash() []byte {
// 	buf := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(buf, t.Nonce)
// 	return buf
// }


// func InitConsensus(roomID string) (error, *hbbft.HoneyBadger) {

	
	
	


// 	// Any new request is processed as a transaction

// 	//
	

// 	// Add Yggdrasil peers

// 	// transport := &NetworkTransport{net: net}

// 	// Use Raft for small networks (2-3 nodes)
// 	// if len(peers) < 4 {
// 	// 	fmt.Println("Using Raft consensus for small network")
// 	// 	// Initialize Raft configuration
// 	// 	config := raft.DefaultConfig()
// 	// 	config.LocalID = raft.ServerID("1") // Use "1" as our ID

// 	// 	// Create Raft transport
// 	// 	raftTransport := &RaftTransport{transport: transport}

// 	// 	// Initialize Raft
// 	// 	var err error
// 	// 	logStore := raft.NewInmemStore()
// 	// 	stableStore := raft.NewInmemStore()
// 	// 	snapshotStore := raft.NewInmemSnapshotStore()
// 	// 	raftNode, err = raft.NewRaft(config, &FSM{}, logStore, stableStore, snapshotStore, raftTransport)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}

// 	// 	// Bootstrap the cluster
// 	// 	if len(peers) == 1 {
// 	// 		configuration := raft.Configuration{
// 	// 			Servers: []raft.Server{
// 	// 				{
// 	// 					ID:      config.LocalID,
// 	// 					Address: raft.ServerAddress(peers[0].IP),
// 	// 				},
// 	// 			},
// 	// 		}
// 	// 		raftNode.BootstrapCluster(configuration)
// 	// 	}
// 	// } else {
// 		// // Use Honey Badger BFT for larger networks
// 		// fmt.Println("Using Honey Badger BFT consensus")

// 		// // Convert IP addresses to uint64 IDs
// 		// // nodeIDs := make([]uint64, len(peers))
// 		// // for i := range peers {
// 		// // 	nodeIDs[i] = uint64(i + 1)
// 		// // }

// 		// cfg := hbbft.Config{
// 		// 	N:         len(peers),
// 		// 	F:         (len(peers) - 1) / 3,
// 		// 	ID:        uint64(1), // Use 1 as our ID
// 		// 	Nodes:     peerIDs,
// 		// 	BatchSize: 100,
// 		// }

// 		// HB := hbbft.NewHoneyBadger(cfg)

// 		// Handle incoming messages
// 		// go func() {
// 		// 	fmt.Printf("Node %d: Starting message handler\n", cfg.ID)
// 		// 	for msg := range transport.Recv() {
// 		// 		fmt.Printf("Node %d: Processing message: %v\n", cfg.ID, msg)
// 		// 		if err := HB.HandleMessage(0, 0, &hbbft.ACSMessage{Payload: msg}); err != nil {
// 		// 			fmt.Printf("Error handling message: %v\n", err)
// 		// 		}
// 		// 	}
// 		// }()
// 	// }

// 	return nil, HB
// }





// func InitConsensus(lenNodes int, peerIDs []uint64 ) {
// 	var (
// 		nodes = MakeNetwork(lenNodes, peerIDs)
// 	)
// 	for _, node := range nodes {
// 		go node.run()
// 		go func(node *Node) {
// 			if err := node.HB.Start(); err != nil {
// 				log.Fatal(err)
// 			}
// 			for _, msg := range node.HB.Messages() {
// 				messages <- message{node.ID, msg}
// 			}
// 		}(node)
// 	}

// 	// handle the relayed transactions.
// 	go func() {
// 		for tx := range relayCh {
// 			for _, node := range nodes {
// 				node.addTransactions(tx)
// 			}
// 		}
// 	}()

// 	for {
// 		msg := <-messages
// 		node := nodes[msg.payload.To]
// 		switch t := msg.payload.Payload.(type) {
// 		case hbbft.HBMessage:
// 			if err := node.hb.HandleMessage(msg.from, t.Epoch, t.Payload.(*hbbft.ACSMessage)); err != nil {
// 				log.Fatal(err)
// 			}
// 			for _, msg := range node.hb.Messages() {
// 				messages <- message{node.id, msg}
// 			}
// 		}
// 	}
// }

// func AddTransaction(data string) {
// 	fmt.Printf("Adding transaction: %s\n", data)
// 	if HB != nil {
// 		// Use Honey Badger BFT
// 		tx := Transaction{Data: data}
// 		HB.AddTransaction(tx)
// 	} else if raftNode != nil {
// 		// Use Raft
// 		raftNode.Apply([]byte(data), time.Second)
// 	}
// }




// func StartConsensus() {
// 	fmt.Println("Starting consensus engine")
// 	if HB != nil {
// 		// Start Honey Badger BFT in a goroutine
// 		go func() {
// 			HB.Start()
// 		}()
// 	}
// 	// Raft starts automatically
// }

// func GetOutputs() map[uint64][]Transaction {
// 	fmt.Println("Getting outputs")
// 	outputs := make(map[uint64][]Transaction)

// 	if HB != nil {
// 		// Get Honey Badger BFT outputs
// 		for epoch, txs := range HB.Outputs() {
// 			var transactions []Transaction
// 			for _, tx := range txs {
// 				if t, ok := tx.(Transaction); ok {
// 					transactions = append(transactions, t)
// 				}
// 			}
// 			outputs[epoch] = transactions
// 		}
// 	} else if raftNode != nil {
// 		// Get Raft outputs (simplified for demo)
// 		outputs[0] = []Transaction{{Data: "Raft consensus output"}}
// 	}

// 	return outputs
// }

// // RaftTransport implements raft.Transport
// type RaftTransport struct {
// 	transport *NetworkTransport
// }

// func (t *RaftTransport) AppendEntriesPipeline(id raft.ServerID, target raft.ServerAddress) (raft.AppendPipeline, error) {
// 	return nil, nil
// }

// func (t *RaftTransport) DecodePeer(peer []byte) raft.ServerAddress {
// 	return raft.ServerAddress(string(peer))
// }

// func (t *RaftTransport) EncodePeer(id raft.ServerID, peer raft.ServerAddress) []byte {
// 	return []byte(peer)
// }

// func (t *RaftTransport) InstallSnapshot(id raft.ServerID, target raft.ServerAddress, args *raft.InstallSnapshotRequest, resp *raft.InstallSnapshotResponse, data io.Reader) error {
// 	return nil
// }

// func (t *RaftTransport) SetHeartbeatHandler(cb func(rpc raft.RPC)) {
// }

// func (t *RaftTransport) TimeoutNow(id raft.ServerID, target raft.ServerAddress, args *raft.TimeoutNowRequest, resp *raft.TimeoutNowResponse) error {
// 	return nil
// }

// func (t *RaftTransport) Consumer() <-chan raft.RPC {
// 	// Implement RPC consumer
// 	return make(chan raft.RPC)
// }

// func (t *RaftTransport) LocalAddr() raft.ServerAddress {
// 	return raft.ServerAddress("localhost")
// }

// func (t *RaftTransport) AppendEntries(id raft.ServerID, target raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
// 	// Implement append entries
// 	return nil
// }

// func (t *RaftTransport) RequestVote(id raft.ServerID, target raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
// 	// Implement request vote
// 	return nil
// }

// // FSM implements raft.FSM
// type FSM struct{}

// func (f *FSM) Apply(log *raft.Log) interface{} {
// 	return nil
// }

// func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
// 	return nil, nil
// }

// func (f *FSM) Restore(io.ReadCloser) error {
// 	return nil
// }
