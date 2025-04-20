package derivationFunctions

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// DeriveIPAddressFromPublicKey derives the IP address from a given public key in Yggdrasil
func DeriveIPAddressFromPublicKey(pubKeyHex string) (string, error) {
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", err
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid public key size")
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	addr := address.AddrForKey(pubKey)
	ip := net.IP(addr[:]).String()

	return ip, nil
}

