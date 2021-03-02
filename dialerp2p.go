// ..

package prnm

import (
	"context"
	"sync"
	"time"

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

	mutex sync.RWMutex
	peers map[wallet.AddrKey]string
}

// NewTCPDialerP2P ...a
func NewTCPDialerP2P(defaultTimeout time.Duration, host host.Host) *DialerP2P {
	log.Println("go-wrapper, dialerp2p.go, NewTCPDialerP2P, 1")
	return &DialerP2P{myHost: host}
}

func (d *DialerP2P) get(key wallet.AddrKey) (string, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	peerID, ok := d.peers[key]
	return peerID, ok
}

// Dial implements Dialer.Dial().
func (d *DialerP2P) Dial(ctx context.Context, addr wire.Address) (wirenet.Conn, error) {
	log.Println("go-wrapper, dialerp2p.go, Dial, 1")

	peerID, ok := d.get(wallet.Key(addr))
	if !ok {
		return nil, errors.New("peer not found")
	}
	log.Println("go-wrapper, dialerp2p.go, Dial, 1.5")

	/* Generate Peer ID from secret key of alice
	sk := "0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9" // secret key of alice
	data, err := crypto.HexToECDSA(sk[2:])
	if err != nil {
		panic(err)
	}
	prvKey, err := cry.UnmarshalSecp256k1PrivateKey(data.X.Bytes())
	if err != nil {
		panic(err)
	}
	pubKey := prvKey.GetPublic()
	anotherClientID, err := peer.IDFromPublicKey(pubKey) */

	anotherClientID, err := peer.Decode(peerID)
	if err != nil {
		return nil, errors.New("peer id is not valid")
	}

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
		return nil, errors.Wrap(err, "failed to dial peer")
	}
	log.Println("go-wrapper, dialerp2p.go, Dial, Connected to another Client!")

	log.Println("go-wrapper, dialerp2p.go, Dial, 4")
	return wirenet.NewIoConn(s), nil
}

// Close ..a
func (d *DialerP2P) Close() error {
	log.Println("go-wrapper, dialerp2p.go, Close, 1")

	err := d.myHost.Close()
	return err
}

// Register registers a libp2p peer id for a peer address.
func (d *DialerP2P) Register(addr wire.Address, peerID string) {
	log.Println("go-wrapper, dialerp2p.go, Register, 1")
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.peers[wallet.Key(addr)] = peerID
	log.Println("go-wrapper, dialerp2p.go, Register, 2")
}
