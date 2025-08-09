package derivationFunctions

import (
	"crypto/ed25519"
	"encoding/hex"
	"net"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// DeriveIPAddressFromPublicKey derives the IP address from a given public key in Yggdrasil
func DeriveIPAddressFromPublicKey(pubKeyHex string) string {
    pubKeyBytes, err := hex.DecodeString(pubKeyHex)
    if err != nil {
        return "invalid public key format"
    }

    if len(pubKeyBytes) != ed25519.PublicKeySize {
        return "invalid public key size"
    }

    pubKey := ed25519.PublicKey(pubKeyBytes)
    addr := address.AddrForKey(pubKey)
    ip := net.IP(addr[:]).String()

    return ip
}

