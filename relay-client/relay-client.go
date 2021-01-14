package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	network "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

//-------------

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
	client := flag.Bool("cli", false, "Client")
	relayInfoID := flag.String("id", "", "ID")
	relayInfoAddrs := flag.String("addr", "", "ADDRS")
	// standAlone := flag.Bool("std", false, "Non Relay Connection Client")
	flag.Parse()

	fmt.Println("Fetching IP4...")
	//Get own ipv4 IP
	res, _ := http.Get("https://api.ipify.org")
	ip, _ := ioutil.ReadAll(res.Body)
	fmt.Println("Received IP")

	//Building Client or Relay
	if *client {
		//Client muss die Mutliaddr vom Relay wieder zusammen setzen
		splitS := strings.Split(*relayInfoAddrs, "_")
		var tempAddr [6]ma.Multiaddr

		for i, s := range splitS {
			temp, err := ma.NewMultiaddr(s)

			if err != nil {
				panic(err)
			} else {
				tempAddr[i] = temp
			}

		}

		addrs := []ma.Multiaddr{tempAddr[0], tempAddr[1], tempAddr[2],
			tempAddr[3], tempAddr[4], tempAddr[5]}

		//Client baut die ID vom Relay auf
		id, err := peer.Decode(*relayInfoID)
		if err != nil {
			panic(err)
		}

		//relayInfo
		relayInfo := peer.AddrInfo{
			ID:    id,
			Addrs: addrs,
		}

		//Client Erstellung
		h1, err := libp2p.New(
			context.Background(),
			libp2p.EnableRelay(),
			//libp2p.EnableAutoRelay(),
		)
		if err != nil {
			panic(err)
		}

		h1ID := h1.ID()

		fmt.Println("Connecting to Relay...")
		if err := h1.Connect(context.Background(), relayInfo); err != nil {
			panic(err)
		}

		//Setup protocol handler
		h1.SetStreamHandler("/client", handleStream)

		//IP from Relay Setup
		ipRunclean := relayInfo.Addrs[0]
		//fmt.Println(ipRunclean)
		ipRclean := strings.Split(ipRunclean.String(), "/")
		ipR := ipRclean[2]
		//fmt.Println(ipR)

		//Convert public IP into strings
		ipS := string(ip)
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println("Successful!")

		fmt.Println("MY IPV4: ", ipS)
		fmt.Println("RELAY IPV4: ", ipR)

		//GET Port
		// Let's get the actual TCP port from our listen multiaddr, in case we're using 0 (default; random available port).
		/*var port string
		for _, la := range relayInfo.Network().ListenAddresses() {
			if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
				port = p
				break
			}
		}
		*/
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

		fullAddr := "/ip4/" + ipR + "/tcp/" + port + "/p2p/" + relayInfo.ID.Pretty() + "/p2p-circuit/p2p/" + h1ID.Pretty()

		//Wait for IP from different User OR connect with IP
		fmt.Println("--------------------------")
		fmt.Println("My ID to connect to: ", h1ID.Pretty())
		fmt.Println("My Address to connect to: ", fullAddr)
		fmt.Println("--------------------------")
		fmt.Println("Waiting for Connection...")
		fmt.Print("OR Connect to ID: ")

		//Scan ID input
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
			relayaddr, err := ma.NewMultiaddr("/ip4/" + ipR + "/tcp/" + port + "/p2p/" + relayInfo.ID.Pretty() + "/p2p-circuit/p2p/" + textDec.Pretty())
			if err != nil {
				panic(err)
			}

			//Redialing hacked
			h1.Network().(*swarm.Swarm).Backoff().Clear(textDec)
			anotherClientInfo := peer.AddrInfo{
				ID:    textDec,
				Addrs: []ma.Multiaddr{relayaddr},
			}
			if err := h1.Connect(context.Background(), anotherClientInfo); err != nil {
				panic(err)
			}

			//Connecting
			s, err := h1.NewStream(context.Background(), anotherClientInfo.ID, "/client")
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

	} else {
		//*********************
		//**STANDALONE CLIENT**
		//*********************
		h1, err := libp2p.New(
			context.Background(),
			libp2p.EnableRelay(),
			//libp2p.EnableAutoRelay(),
		)
		if err != nil {
			panic(err)
		}

		fmt.Println("Use full adress to connect: ")
		//Scan input
		var text string
		fmt.Scanln(&text)

		//relayAddr ADDRESSE FÜR ANOTHERCLIENT
		relayaddr, err := ma.NewMultiaddr(text)
		if err != nil {
			panic(err)
		}

		//Extract ID from fullAddr
		textA := strings.Split(text, "/")
		textDec, err := peer.Decode(textA[len(textA)-1])
		if err != nil {
			panic(err)
		}

		//Redialing hacked
		h1.Network().(*swarm.Swarm).Backoff().Clear(textDec)
		anotherClientInfo := peer.AddrInfo{
			ID:    textDec,
			Addrs: []ma.Multiaddr{relayaddr},
		}
		if err := h1.Connect(context.Background(), anotherClientInfo); err != nil {
			panic(err)
		}

		//Connecting
		s, err := h1.NewStream(context.Background(), anotherClientInfo.ID, "/client")
		if err != nil {
			fmt.Println("Not working: ", err)
			return
		}

		fmt.Println("Connected to AnotherClient!")

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		go writeData(rw)
		go readData(rw)

	}

	//Everything is built, wait for messages or peers(?)
	<-make(chan struct{})
}
