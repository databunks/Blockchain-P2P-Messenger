package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
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
var blockchains map[string][]Block // roomID -> []Block

func init() {
	blockchains = make(map[string][]Block)
}

// CalculateHash is a simple SHA256 hashing function
func CalculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + block.Data + block.PrevHash
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
	if len(newBlocks) > len(blockchains[roomID]) {
		blockchains[roomID] = newBlocks
		return SaveBlockchainToFile(roomID)
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

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(blockchains[roomID])
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
			blockchains[roomID] = []Block{genesisBlock}
			return SaveBlockchainToFile(roomID)
		}
		return err
	}
	defer file.Close()

	var blocks []Block
	if err := json.NewDecoder(file).Decode(&blocks); err != nil {
		return err
	}
	blockchains[roomID] = blocks
	return nil
}

// AddBlock adds a new block with the given data to the blockchain
func AddBlock(data string, roomID string) error {
	if err := LoadBlockchainFromFile(roomID); err != nil {
		return err
	}

	lastBlock := blockchains[roomID][len(blockchains[roomID])-1]
	newBlock, err := GenerateBlock(lastBlock, data)
	if err != nil {
		return err
	}

	blockchains[roomID] = append(blockchains[roomID], newBlock)
	return SaveBlockchainToFile(roomID)
}

// GetBlockchain returns the blockchain for a specific room
func GetBlockchain(roomID string) []Block {
	LoadBlockchainFromFile(roomID)
	if chain, exists := blockchains[roomID]; exists {
		return chain
	}
	return []Block{}
}
