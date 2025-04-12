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
var Blockchain []Block

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

	// Save the updated blockchain to file
	if err := SaveBlockchainToFile(); err != nil {
		return Block{}, err
	}

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
func ReplaceChain(newBlocks []Block) error {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
		return SaveBlockchainToFile()
	}
	return nil
}

// SaveBlockchainToFile saves the current blockchain to a file
func SaveBlockchainToFile() error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(dataDir, "blockchain.json"),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(Blockchain)
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
func LoadBlockchainFromFile() error {
	file, err := os.Open(filepath.Join(dataDir, "blockchain.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with genesis block if file doesn't exist
			genesisBlock := GenerateGenesisBlock()
			Blockchain = []Block{genesisBlock}
			return SaveBlockchainToFile()
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&Blockchain)
}

// AddBlock adds a new block with the given data to the blockchain
func AddBlock(data string) error {
	if err := LoadBlockchainFromFile(); err != nil {
		return err
	}

	lastBlock := Blockchain[len(Blockchain)-1]
	newBlock, err := GenerateBlock(lastBlock, data)
	if err != nil {
		return err
	}

	Blockchain = append(Blockchain, newBlock)
	return SaveBlockchainToFile()
}
