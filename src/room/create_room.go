package room

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Room struct {
	ID        string
	Name      string
	Members   []string // List of member public keys
	CreatedAt int64
}

// CreateRoom creates a new chat room
func CreateRoom(name string) (*Room, error) {
	// Generate a random room ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}

	room := &Room{
		ID:        hex.EncodeToString(idBytes),
		Name:      name,
		Members:   make([]string, 0),
		CreatedAt: time.Now().Unix(),
	}

	return room, nil
}
