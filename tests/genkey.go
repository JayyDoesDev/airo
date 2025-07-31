package main

import (
	"fmt"

	"github.com/jayydoesdev/airo/bot/cryptography"
)

func GenKey() {
	key, err := cryptography.GenerateKey(32)
	if err != nil {
		fmt.Println("failed to get key")
		return
	}

	fmt.Println(key)
}

func main() {
	go func() {
		GenKey()
	}()

	select {}
}
