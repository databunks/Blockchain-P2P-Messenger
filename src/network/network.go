package network

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"
)

type Message struct {
	From    string
	To      string
	Payload []byte
}

type Network struct {
	nodeID  string
	peers   map[string]string // nodeID -> address
	conns   map[string]net.Conn
	msgChan chan Message
	mu      sync.Mutex
}

func NewNetwork(nodeID string) *Network {
	return &Network{
		nodeID:  nodeID,
		peers:   make(map[string]string),
		conns:   make(map[string]net.Conn),
		msgChan: make(chan Message, 100),
	}
}

// GetPort returns the current port this node is listening on
func (n *Network) GetPort() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	_, port, err := net.SplitHostPort(n.peers[n.nodeID])
	if err != nil {
		return 0
	}
	p, _ := net.LookupPort("tcp", port)
	return p
}

func (n *Network) AddPeer(nodeID string, addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// If this is our own node ID, ensure we're using localhost
	if nodeID == n.nodeID {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return
		}
		n.peers[nodeID] = fmt.Sprintf("localhost:%s", port)
	} else {
		n.peers[nodeID] = addr
	}
}

// findAvailablePort tries to find an available port starting from the given port
func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found between %d and %d", startPort, startPort+100)
}

func (n *Network) Start(port int) error {
	// Find an available port
	availablePort, err := findAvailablePort(port)
	if err != nil {
		return err
	}

	// If we had to change the port, only update our own address
	if availablePort != port {
		fmt.Printf("Port %d was in use, using port %d instead\n", port, availablePort)
		// Update only our own address in the peers map
		n.peers[n.nodeID] = fmt.Sprintf("localhost:%d", availablePort)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", availablePort))
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go n.handleConnection(conn)
		}
	}()

	// Wait for all peers to come online before connecting
	fmt.Printf("Node %s: Waiting for peers to come online...\n", n.nodeID)
	for id, addr := range n.peers {
		if id != n.nodeID {
			for {
				conn, err := net.Dial("tcp", addr)
				if err == nil {
					n.conns[id] = conn
					fmt.Printf("Node %s: Connected to peer %s at %s\n", n.nodeID, id, addr)
					go n.handleConnection(conn)
					break
				}
				fmt.Printf("Node %s: Failed to connect to peer %s at %s, retrying...\n", n.nodeID, id, addr)
				time.Sleep(time.Second)
			}
		}
	}
	fmt.Printf("Node %s: All peers are online!\n", n.nodeID)

	return nil
}

func (n *Network) handleConnection(conn net.Conn) {
	decoder := gob.NewDecoder(conn)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			continue
		}
		n.msgChan <- msg
	}
}

func (n *Network) Send(to string, data []byte) error {
	n.mu.Lock()
	conn, ok := n.conns[to]
	n.mu.Unlock()

	if !ok {
		return fmt.Errorf("no connection to node %s", to)
	}

	msg := Message{
		From:    n.nodeID,
		To:      to,
		Payload: data,
	}

	encoder := gob.NewEncoder(conn)
	return encoder.Encode(msg)
}

func (n *Network) Receive() <-chan Message {
	return n.msgChan
}
