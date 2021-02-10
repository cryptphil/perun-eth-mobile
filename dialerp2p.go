// ..

package prnm

import (
	"context"
	"encoding/binary"
	mrand "math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	"perun.network/go-perun/log"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
	wirenet "perun.network/go-perun/wire/net"
)

// DialerP2P ...l
type DialerP2P struct {
	myHost host.Host
}

// NewTCPDialerP2P ...a
func NewTCPDialerP2P(defaultTimeout time.Duration, host host.Host) *DialerP2P {
	log.Println("go-wrapper, dialerp2p.go, NewTCPDialerP2P, 1")
	return &DialerP2P{myHost: host}
}

// Dial ...
func (d *DialerP2P) Dial(ctx context.Context, addr wire.Address) (wirenet.Conn, error) {
	log.Println("go-wrapper, dialerp2p.go, Dial, 1")
	log.Println("go-wrapper, dialerp2p.go, Dial, Wire Addresses looks like ", addr.String())
	log.Println("go-wrapper, dialerp2p.go, Dial, Wallet Key From Wire Addresses looks like ", wallet.Key(addr))

	// Generate Peer ID From Wire Address

	data := binary.BigEndian.Uint64(addr.Bytes())
	r := mrand.New(mrand.NewSource(int64(data)))
	_, pubKey, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	x, err := peer.IDFromPublicKey(pubKey)
	log.Println("go-wrapper, dialerp2p.go, Dial, IDFromPublickey", x)

	var anotherClientID peer.ID = x

	fullAddr := serverAddr + "/p2p/" + serverID + "/p2p-circuit/p2p/" + anotherClientID.Pretty()
	AnotherClientMA, err := ma.NewMultiaddr(fullAddr)
	if err != nil {
		panic(err)
	}

	log.Println("go-wrapper, dialerp2p.go, Dial, 2")
	//Redialing hacked
	d.myHost.Network().(*swarm.Swarm).Backoff().Clear(anotherClientID)
	anotherClientInfo := peer.AddrInfo{
		ID:    anotherClientID,
		Addrs: []ma.Multiaddr{AnotherClientMA},
	}
	if err := d.myHost.Connect(context.Background(), anotherClientInfo); err != nil {
		panic(err)
	}

	log.Println("go-wrapper, dialerp2p.go, Dial, 3")
	//Connecting
	s, err := d.myHost.NewStream(context.Background(), anotherClientInfo.ID, "/client")
	if err != nil {
		return nil, errors.Wrap(err, "Not working")
	}
	log.Println("go-wrapper, dialerp2p.go, Dial, Connected to another Client!")

	//reader := bufio.NewReader(s)
	//writer := bufio.NewWriter(s)
	//rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	//var rwc io.ReadWriteCloser = &ClosableBufio{*reader, *writer}

	log.Println("go-wrapper, dialerp2p.go, Dial, 4")
	return wirenet.NewIoConn(s), nil
}

// Close ..a
func (d *DialerP2P) Close() error {
	log.Println("go-wrapper, dialerp2p.go, Close, 1")

	err := d.myHost.Close()
	return err
}

// Register ..a
func (d *DialerP2P) Register(addr wire.Address, address string) {
	log.Println("go-wrapper, dialerp2p.go, Register, 1")
	log.Println("go-wrapper, dialerp2p.go, Register, Wire Addresses looks like ", addr.String())
	log.Println("go-wrapper, dialerp2p.go, Register, Wallet Key From Wire Addresses looks like ", wallet.Key(addr))
	log.Println("go-wrapper, dialerp2p.go, Register, only address", address)
}
