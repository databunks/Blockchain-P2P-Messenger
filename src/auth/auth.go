package auth

import (
	"blockchain-p2p-messenger/src/blockchain"
	"fmt"
	"strings"
)

var administrators []string

func init() {
	administrators = make([]string, 0)
}

// loadAdminsFromBlockchain scans the blockchain to build the admin list
func loadAdminsFromBlockchain(chain []blockchain.Block) {
	administrators = make([]string, 0) // Reset the list
	for _, block := range chain {
		if strings.HasPrefix(block.Data, "ADMIN_ADD:") {
			publicKey := strings.TrimPrefix(block.Data, "ADMIN_ADD:")
			administrators = append(administrators, publicKey)
		} else if strings.HasPrefix(block.Data, "ADMIN_REMOVE:") {
			publicKey := strings.TrimPrefix(block.Data, "ADMIN_REMOVE:")
			// Remove the admin from the list
			var newAdmins []string
			for _, admin := range administrators {
				if admin != publicKey {
					newAdmins = append(newAdmins, admin)
				}
			}
			administrators = newAdmins
		}
	}
}

// AddAdmin adds a new administrator
func AddAdmin(publicKey string, chain []blockchain.Block, roomID string) error {
	// Load current admin list from blockchain
	loadAdminsFromBlockchain(chain)

	// Check if already an admin
	if IsAdmin(publicKey) {
		return nil // Already an admin
	}

	// Record in blockchain
	adminData := fmt.Sprintf("ADMIN_ADD:%s", publicKey)
	return blockchain.AddBlock(adminData, roomID)
}

// RemoveAdmin removes an administrator
func RemoveAdmin(publicKey string, chain []blockchain.Block, roomID string) error {
	// Load current admin list from blockchain
	loadAdminsFromBlockchain(chain)

	// Check if is an admin
	if !IsAdmin(publicKey) {
		return nil // Not an admin
	}

	// Record in blockchain
	adminData := fmt.Sprintf("ADMIN_REMOVE:%s", publicKey)
	return blockchain.AddBlock(adminData, roomID)
}

// IsAdmin checks if a public key belongs to an administrator
func IsAdmin(publicKey string) bool {
	for _, admin := range administrators {
		if admin == publicKey {
			return true
		}
	}
	return false
}
