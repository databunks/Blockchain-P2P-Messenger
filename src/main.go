package main

import (
	"blockchain-p2p-messenger/src/auth"
	"fmt"
)

func main() {
	fmt.Println("Hello, World!")
	auth.AddToAllowedList("127.0.0.1")
	fmt.Println(auth.AllowedList)
}
