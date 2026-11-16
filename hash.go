package main

import (
	"fmt"

	"elayan.local/desa/internal/security"
)

func main() {
	hash, err := security.HashPassword("Kamehame")
	if err != nil {
		panic(err)
	}

	fmt.Println(hash)
}
