package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	network "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

func handleStream(s network.Stream) {
	log.Println()
	log.Println("Got a new Stream!")

	//Create non blocking read-writes
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)

	//streams 's' will stay open until you close it (or the other side)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}
	}
}
func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
	}
}

func main() {
	relayInfoID := flag.String("id", "", "ID")
	relayInfoAddrs := flag.String("addr", "", "ADDRS")
	flag.Parse()

	// Parse Relay Peer ID
	id, err := peer.Decode(*relayInfoID)
	if err != nil {
		panic(err)
	}
	// Parse Relay Multiadress
	tmp, err := ma.NewMultiaddr(*relayInfoAddrs)
	if err != nil {
		panic(err)
	}
	addrs := []ma.Multiaddr{tmp}
	// Build now relay's AddrInfo
	relayInfo := peer.AddrInfo{
		ID:    id,
		Addrs: addrs,
	}

	// Creates a new random RSA key pair for this host.
	r := rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// Construct a new libp2p client for our relay-server.
	// Background()		-
	// EnableRelay() 	-
	// Identity(prvKey)	- Use a RSA private key to generate the ID of the host.
	client, err := libp2p.New(
		context.Background(),
		libp2p.EnableRelay(),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}

	// Connect to relay server
	fmt.Println("Connecting to Relay...")
	if err := client.Connect(context.Background(), relayInfo); err != nil {
		panic(err)
	}
	fmt.Println(".... Successful!")

	// Setup protocol handler
	client.SetStreamHandler("/client", handleStream)

	// Search for a local available port.
	var port string
	for _, la := range relayInfo.Addrs {
		if p, err := la.ValueForProtocol(ma.P_TCP); err == nil {
			port = p
			break
		}
	}
	if port == "" {
		panic("was not able to find actual local port")
	}

	// Build own full address
	fullAddr := relayInfo.Addrs[0].String() + "/p2p/" + relayInfo.ID.Pretty() + "/p2p-circuit/p2p/" + client.ID().Pretty()

	//Wait for IP from different User OR connect with IP
	fmt.Println("--------------------------")
	fmt.Println("My ID to connect to: ", client.ID().Pretty())
	fmt.Println("My Address to connect to: ", fullAddr)
	fmt.Println("--------------------------")
	fmt.Println("Waiting for Connection...")
	fmt.Print("OR Connect to ID: ")

	// Scan ID input
	var text string
	fmt.Scanln(&text)

	decodable := true
	//Decode ID
	textDec, err := peer.Decode(text)
	if err != nil {
		decodable = false
	}

	if decodable {
		//relayAddr ADDRESSE FÜR ANOTHERCLIENT
		relayaddr, err := ma.NewMultiaddr(relayInfo.Addrs[0].String() + "/p2p/" + relayInfo.ID.Pretty() + "/p2p-circuit/p2p/" + textDec.Pretty())
		if err != nil {
			panic(err)
		}

		//Redialing hacked
		client.Network().(*swarm.Swarm).Backoff().Clear(textDec)
		anotherClientInfo := peer.AddrInfo{
			ID:    textDec,
			Addrs: []ma.Multiaddr{relayaddr},
		}
		if err := client.Connect(context.Background(), anotherClientInfo); err != nil {
			panic(err)
		}

		//Connecting
		s, err := client.NewStream(context.Background(), anotherClientInfo.ID, "/client")
		if err != nil {
			fmt.Println("Not working: ", err)
			return
		}
		fmt.Println("Connected to another Client!")

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		go writeData(rw)
		go readData(rw)
	}

	<-make(chan struct{})
}
