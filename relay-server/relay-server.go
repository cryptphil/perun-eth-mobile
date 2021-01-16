package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	mrand "math/rand"
	"net/http"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	circuit "github.com/libp2p/go-libp2p-circuit"
)

func main() {
	// Display help menu if you the flag "help"
	help := flag.Bool("help", false, "Display help")
	flag.Parse()
	if *help {
		fmt.Printf("This program is a relay server for the perun-network using libp2p\n\n")
		fmt.Println("Usage: Run './relay-server' to start the relay-server.")
		os.Exit(0)
	}

	// Use the constant number 1234 as the randomness source to generate the peer ID.
	// This will always generate the same host ID on multiple executions, unless you change the number.
	// If you don't want that use r := rand.Reader instead
	r := mrand.New(mrand.NewSource(int64(1234)))
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// Fetch own _public_ IPv4 address using ipify api
	log.Println("Fetching own public IPv4 address...")
	res, _ := http.Get("https://api.ipify.org")
	ip, _ := ioutil.ReadAll(res.Body)
	log.Println("Received own public IPv4 address")

	// Build host multiaddress
	// 0.0.0.0 will listen on any interface device.
	sourcePort := 5574
	sourceMultiAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", sourcePort))

	// Construct a new libp2p host: our relay server.
	// EnableRelay(circuit.OptHop) 	-
	// ListenAddrs(sourceMultiAddr)	-
	// Identity(prvKey)				- Use a RSA private key to generate the ID of the host.
	relay, err := libp2p.New(context.Background(),
		libp2p.EnableRelay(circuit.OptHop),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		log.Fatalln(err)
		// panic(err)
	}

	// AddrInfo	-
	// ID 		- unique Peer ID of the relay server
	// Addr		-
	relayinfo := peer.AddrInfo{
		ID:    relay.ID(),
		Addrs: relay.Addrs(),
	}

	// Get the actual TCP port from our listen multiaddr, in case we're using 0 (default; random available port).
	var port string
	for _, la := range relay.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(ma.P_TCP); err == nil {
			port = p
			break
		}
	}
	if port == "" {
		// log.Fatalln("was not able to find actual local port")
		panic("was not able to find actual local port")
	}

	// Display this string if no client has connected yet and every time a new client has connected
	sb := "\n\n\n\n" +
		"\n------------------------" +
		"\nListening on port: " + port +
		"\nRelay Station is now running..." +
		"\n------------------------" +
		"\nRun: 'relay-client.exe -id " + peer.Encode(relayinfo.ID) + " -addr " +
		"/ip4/" + string(ip) + "/tcp/" + fmt.Sprint(sourcePort) +
		"' on another console to get a client to connect to this relay server." +
		"\n------------------------"
	log.Println(sb)

	// TODO: How to get which clients are still connected?
	// REACT TO NEW PEERS
	var store peer.IDSlice
	store = relay.Peerstore().Peers()
	fmt.Println("Known Peers: 0")
	for {
		same := true
		found := false

		for _, store := range store {
			for _, peer := range relay.Peerstore().Peers() {
				if store == peer {
					found = true
					break
				}
			}
			if !found {
				same = false
				break
			}
			found = false
		}

		if same {
			if len(store) == len(relay.Peerstore().Peers()) {
				continue
			}
		}

		store = relay.Peerstore().Peers()
		fmt.Println(sb+"Known Peers: ", len(store)-1)
		for _, p := range store {
			fmt.Println(p)
		}
	}
}
