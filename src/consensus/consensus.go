package consensus

import (
	"blockchain-p2p-messenger/src/blockchain"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/anthdm/hbbft"
)

const (
	batchSize = 1
	numCores  = 4
)

type message struct {
	from    uint64
	payload hbbft.MessageTuple
}

// Server represents the local node.
type Server struct {
	ID          uint64
	HB          *hbbft.HoneyBadger
	transport   hbbft.Transport
	rpcCh       <-chan hbbft.RPC
	lock        sync.RWMutex
	mempool     map[string]*Transaction
	totalCommit int
	start       time.Time
}

var (
	txDelay  = (3 * time.Millisecond) / numCores
	messages = make(chan message, 1024*1024)
	relayCh  = make(chan *Transaction, 1024*1024)
)

func InitializeConsensus(lenNodes int, peerIDs []uint64) []*Server {
	var (
		nodes = MakeNetwork(lenNodes, peerIDs)
	)


	go func(){
		for _, node := range nodes {
			go node.commitLoop()
			go func(node *Server) {
				
				if err := node.HB.Start(); err != nil {
					log.Fatal(err)
				}
				for _, msg := range node.HB.Messages() {
					messages <- message{node.ID, msg}
					
				}

			}(node)
		}
		
		// handle the relayed transactions.
		go func() {
			for tx := range relayCh {
				for _, node := range nodes {
					node.addTransactions(tx)
				}
			}

		}()

		for {
			msg := <-messages
			node := nodes[msg.payload.To]
			switch t := msg.payload.Payload.(type) {
			case hbbft.HBMessage:
				if err := node.HB.HandleMessage(msg.from, t.Epoch, t.Payload.(*hbbft.ACSMessage)); err != nil {
					log.Fatal(err)
				}

				for _, msg := range node.HB.Messages() {
					messages <- message{node.ID, msg}
				}
			}
		}

	}()

	return nodes
}

func (t *Transaction) Hash() []byte {
	data := fmt.Sprintf("%s:%s:%s:%d", t.SenderPublicKey, t.Typeofmessage, t.DigitalSignature, t.Timestamp)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

type Transaction struct {
	SenderPublicKey  string
	Typeofmessage    string
	DigitalSignature string
	Timestamp        uint64
	RoomID 			 string
}

func NewTransaction(senderPublicKey string, typeofmessage string, digitalSignature string, timestamp uint64, roomID string) *Transaction {
	return &Transaction{senderPublicKey, typeofmessage, digitalSignature, timestamp, roomID}
}


func NewServer(id uint64, tr hbbft.Transport, nodes []uint64) *Server {
	hb := hbbft.NewHoneyBadger(hbbft.Config{
		N:         len(nodes),
		ID:        id,
		Nodes:     nodes,
		BatchSize: batchSize,
	})
	return &Server{
		ID:        id,
		transport: tr,
		HB:        hb,
		rpcCh:     tr.Consume(),
		mempool:   make(map[string]*Transaction),
		start:     time.Now(),
	}
}

// Simulate the delay of verifying a transaction.
func (s *Server) VerifyTransaction(tx *Transaction) bool {
	//time.Sleep(txDelay)
	return true
}

func (s *Server) addTransactions(txx ...*Transaction) {
	for _, tx := range txx {
		if s.VerifyTransaction(tx) {
			s.lock.Lock()
			s.mempool[string(tx.Hash())] = tx
			s.lock.Unlock()

			// Add this transaction to the hbbft buffer.
			s.HB.AddTransaction(tx)
			// relay the transaction to all other nodes in the network.
			go func() {
				for i := 0; i < len(s.HB.Nodes); i++ {
					if uint64(i) != s.HB.ID {
						relayCh <- tx
					}
				}
			}()
		}
	}
}

func (s *Server) commitLoop() {
	timer := time.NewTicker(time.Second * 2)
	n := 0
	for {
		select {
		case <-timer.C:
			out := s.HB.Outputs()
			for _, txx := range out {
				for _, tx := range txx {
					hash := tx.Hash()

					fmt.Println("Was here!!")

					txString := fmt.Sprint("%s", tx)
					txString = strings.Split(txString, "%s")[1]
					removeChars := "&{}"

					txString = strings.Map(func(r rune) rune {
						if strings.ContainsRune(removeChars, r) {
							return -1 // skip this rune
						}
						return r
					}, txString)

			
					txSplit := strings.Split(txString, " ")

					blockchainStr := "MESSAGE_ADDED{SenderPublicKey: " + txSplit[0]  +", TypeOfMessage: "+ txSplit[1] + ", DigitalSignature: " + txSplit[2] + ", Timestamp: "  + txSplit[3] + ", RoomID: " + txSplit[4] +"}"
					blockchain.AddBlock(blockchainStr, txSplit[4])

					fmt.Println(tx)
					s.lock.Lock()
					if _, ok := s.mempool[string(hash)]; !ok {
						// Transaction is not in our mempool which implies we
						// need to do verification.
					}
					n++
					delete(s.mempool, string(hash))
					s.lock.Unlock()
				}
			}
			s.totalCommit += n
			//delta := time.Since(s.start)
			if s.ID == 1 {
				// fmt.Println("")
				// fmt.Println("===============================================")
				// fmt.Printf("SERVER (%d)\n", s. ID)
				// fmt.Printf("commited %d transactions over %v\n", s.totalCommit, delta)
				// fmt.Printf("throughput %d TX/s\n", s.totalCommit/int(delta.Seconds()))
				// fmt.Println("===============================================")
				// fmt.Println("")
			}
			n = 0
		}
	}
}



func MakeNetwork(n int, nodeIDs []uint64) []*Server {
	transports := make([]hbbft.Transport, n)
	nodes := make([]*Server, n)
	for i := 0; i < n; i++ {
		transports[i] = hbbft.NewLocalTransport(nodeIDs[i])
		nodes[i] = NewServer(nodeIDs[i], transports[i], nodeIDs)
	}
	ConnectTransports(transports)
	return nodes
}

func ConnectTransports(tt []hbbft.Transport) {
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
