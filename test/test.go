package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	cry "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	serverID   = "QmPyRxsUQfAWR6uYYkSoZQsaM1pra2qpUHE3CMTgrfsTEV"
	serverAddr = "/ip4/77.189.187.162/tcp/5574"
)

func main() {
	// Parse Relay Peer ID
	id, err := peer.Decode(serverID)
	if err != nil {
		panic(err)
	}
	// Parse Relay Multiadress
	tmp, err := ma.NewMultiaddr(serverAddr)
	if err != nil {
		panic(err)
	}
	addrs := []ma.Multiaddr{tmp}
	// Build now relay's AddrInfo
	relayInfo := peer.AddrInfo{
		ID:    id,
		Addrs: addrs,
	}

	sk := "0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a" // secret key of alice
	data, err := crypto.HexToECDSA(sk[2:])
	if err != nil {
		panic(err)
	}

	//prvKey, err := cry.UnmarshalEd25519PrivateKey(data)
	prvKey, err := cry.UnmarshalSecp256k1PrivateKey(data.X.Bytes())
	if err != nil {
		panic(err)
	}
	fmt.Println("Hi")
	log.Println("go-wrapper, client.go, CreateClientHost, 2")
	// Construct a new libp2p client for our relay-server.
	// Background()		-
	// EnableRelay() 	-
	// Identity(prvKey)	- Use  private key to generate the ID of the host.
	client, err := libp2p.New(
		context.Background(),
		libp2p.EnableRelay(),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}
	log.Println("go-wrapper, client.go, ClientID: ", client.ID())

	log.Println("go-wrapper, client.go, CreateClientHost, 3")
	// Connect to relay server
	fmt.Println("Connecting to Relay...")
	if err := client.Connect(context.Background(), relayInfo); err != nil {
		panic(err)
	}
	fmt.Println(".... Successful!")
}
