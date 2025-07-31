package main

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/anthdm/hbbft"
)

type Transaction struct {
	Nonce uint64
}

// Hash implements hbbft.Transaction.
func (t *Transaction) Hash() []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, t.Nonce)
	return buf
}

func newTransaction() *Transaction {
	return &Transaction{rand.Uint64()}
}

func main() {

	uint64Slice := []uint64{67, 1, 99, 101}

	// Create a Config struct with your prefered settings.
	cfg := hbbft.Config{
		// The number of nodes in the network.
		N: 4,
		// Identifier of this node.
		ID: 101,
		// Identifiers of the participating nodes.
		Nodes: uint64Slice,
		// The prefered batch size. If BatchSize is empty, an ideal batch size will
		// be choosen for you.
		BatchSize: 1,
	}

	// Create a new instance of the HoneyBadger engine and pass in the config.
	hb := hbbft.NewHoneyBadger(cfg)


	go func () {
		hb.Start()

		fmt.Println(hb.Messages())
		//fmt.Println("EEEEEE")
		

	}()


	hb.AddTransaction(newTransaction())
	
	time.Sleep(5233200 * time.Millisecond)

	

	// for {
	// 	// Give the protocol enough time to start processing and communicating.
	// 	time.Sleep(500 * time.Millisecond)

	// 	// Check if any messages were produced.
	// 	msgs := hb.Messages()
	// 	if len(msgs) > 0 {
	// 		fmt.Println("Messages:", msgs)
	// 	} else {
	// 		fmt.Println("No messages yet. Waiting for protocol to produce outputs...")
	// 	}
	// }
}
