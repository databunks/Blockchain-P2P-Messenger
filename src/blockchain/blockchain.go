package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const dataDir = "data"

// Block represents each 'item' in the blockchain
type Block struct {
	Index     int    `json:"index"`
	Timestamp string `json:"timestamp"`
	Data      string `json:"data"`
	Hash      string `json:"hash"`
	PrevHash  string `json:"prev_hash"`
}

// Blockchain is a series of validated Blocks
var blockchains sync.Map // roomID -> []Block (thread-safe)

// CalculateHash is a simple SHA256 hashing function
// Note: Timestamp is excluded from hash calculation to ensure identical messages
// generate identical hashes regardless of when they're processed
func CalculateHash(block Block) string {
	// Exclude timestamp from hash calculation for consensus consistency
	record := string(block.Index) + block.Data + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// GenerateBlock creates a new block using previous block's hash
func GenerateBlock(oldBlock Block, Data string) (Block, error) {
	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Data = Data
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = CalculateHash(newBlock)

	return newBlock, nil
}

// IsBlockValid makes sure block is valid by checking index and comparing the hash of the previous block
func IsBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if CalculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// ReplaceChain replaces the current chain with a new one if the new one is longer
func ReplaceChain(newBlocks []Block, roomID string) error {
	if existingChain, exists := blockchains.Load(roomID); exists {
		if existingBlocks, ok := existingChain.([]Block); ok {
			if len(newBlocks) > len(existingBlocks) {
				blockchains.Store(roomID, newBlocks)
				return SaveBlockchainToFile(roomID)
			}
		}
	}
	return nil
}

// SaveBlockchainToFile saves the current blockchain to a file
func SaveBlockchainToFile(roomID string) error {
	roomDir := filepath.Join(dataDir, roomID)
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(roomDir, "blockchain.json"),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if chain, exists := blockchains.Load(roomID); exists {
		if blocks, ok := chain.([]Block); ok {
			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")
			return encoder.Encode(blocks)
		}
	}
	return nil
}

// GenerateGenesisBlock creates the first block in the blockchain
func GenerateGenesisBlock() Block {
	genesisBlock := Block{
		Index:     0,
		Timestamp: time.Now().String(),
		Data:      "Genesis Block",
		PrevHash:  "",
	}
	genesisBlock.Hash = CalculateHash(genesisBlock)
	return genesisBlock
}

// LoadBlockchainFromFile loads the blockchain from a file
func LoadBlockchainFromFile(roomID string) error {
	roomDir := filepath.Join(dataDir, roomID)
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return err
	}

	file, err := os.Open(filepath.Join(roomDir, "blockchain.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with genesis block if file doesn't exist
			genesisBlock := GenerateGenesisBlock()
			blockchains.Store(roomID, []Block{genesisBlock})
			return SaveBlockchainToFile(roomID)
		}
		return err
	}
	defer file.Close()

	var blocks []Block
	if err := json.NewDecoder(file).Decode(&blocks); err != nil {
		return err
	}
	blockchains.Store(roomID, blocks)
	return nil
}

// AddBlock adds a new block with the given data to the blockchain
func AddBlock(data string, roomID string) error {
	if err := LoadBlockchainFromFile(roomID); err != nil {
		return err
	}

	if chain, exists := blockchains.Load(roomID); exists {
		if blocks, ok := chain.([]Block); ok {
			lastBlock := blocks[len(blocks)-1]
			newBlock, err := GenerateBlock(lastBlock, data)
			if err != nil {
				return err
			}

			newBlocks := append(blocks, newBlock)
			blockchains.Store(roomID, newBlocks)
			return SaveBlockchainToFile(roomID)
		}
	}
	return nil
}

// GetBlockchain returns the blockchain for a specific room
func GetBlockchain(roomID string) []Block {
	LoadBlockchainFromFile(roomID)
	if chain, exists := blockchains.Load(roomID); exists {
		if blocks, ok := chain.([]Block); ok {
			return blocks
		}
	}
	return []Block{}
}
