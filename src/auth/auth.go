package auth

import (
	"encoding/hex"
)

var AllowedList = []string{}

var Administrator_Status = false

func AddToAllowedList(ip string) {
	AllowedList = append(AllowedList, ip)
}

// GeneratePublicKeyFromPrivate generates a Yggdrasil public key from a private key
func GeneratePublicKeyFromPrivate(privateKeyHex string) (string, error) {
	// Decode the hex private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", err
	}

	// Create a new private key object
	var privateKey crypto.BoxPrivateKey
	copy(privateKey[:], privateKeyBytes)

	// Generate the public key
	publicKey := privateKey.PublicKey()

	// Convert to hex string
	return hex.EncodeToString(publicKey[:]), nil
}
