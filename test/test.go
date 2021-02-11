package main

import (
	"encoding/hex"
	"fmt"

	cry "github.com/libp2p/go-libp2p-core/crypto"
)

func main() {

	sk := "0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a" // secret key of alice
	data, err := hex.DecodeString(sk[2:] + sk[2:])
	if err != nil {
		panic(err)
	}

	fmt.Println(len(data))
	fmt.Println(len(data[16:]))

	prvKey, err := cry.UnmarshalEd25519PrivateKey(data)
	if err != nil {
		panic(err)
	}
	fmt.Println("Hi")
	fmt.Println(prvKey.GetPublic())
}
