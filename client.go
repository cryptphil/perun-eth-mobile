// Copyright (c) 2021 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"context"

	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/libp2p/go-libp2p"
	cry "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/backend/ethereum/wallet/keystore"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
	"perun.network/go-perun/pkg/sortedkv/leveldb"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire/net"
)

type (
	// Client is a state channel client. It is the central controller to interact
	// with a state channel network. It can be used to propose channels to other
	// channel network peers.
	// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client
	Client struct {
		cfg *Config

		ethClient *ethclient.Client
		client    *client.Client
		persister *keyvalue.PersistRestorer

		wallet  *keystore.Wallet
		onChain wallet.Account

		//dialer *simple.Dialer
		dialer *DialerP2P
		Bus    *net.Bus

		PeerID string // used in libp2p
	}

	// NewChannelCallback wraps a `func(*PaymentChannel)`
	// function pointer for the `Client.OnNewChannel` callback.
	NewChannelCallback interface {
		OnNew(*PaymentChannel)
	}
)

const (
	serverID   = "QmPyRxsUQfAWR6uYYkSoZQsaM1pra2qpUHE3CMTgrfsTEV"
	serverAddr = "/ip4/77.190.27.9/tcp/5574"
)

// GetLibP2PID getted the peer id of the client.
func (c *Client) GetLibP2PID() string {
	return c.PeerID
}

// CreateClientHost connects to a specific relay.
func CreateClientHost(sk string) host.Host {
	log.Println("go-wrapper, client.go, CreateClientHost, 1")

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

	// Create Peer ID from given ESCDA secret key.
	//sk := "0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9" // secret key of alice
	// sk := "0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a" // secret key of bob
	data, err := crypto.HexToECDSA(sk[2:])
	if err != nil {
		panic(err)
	}
	prvKey, err := cry.UnmarshalSecp256k1PrivateKey(data.X.Bytes())
	if err != nil {
		panic(err)
	}

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

	log.Println("go-wrapper, client.go, CreateClientHost, 4")

	// Build own full address
	fullAddr := relayInfo.Addrs[0].String() + "/p2p/" + relayInfo.ID.Pretty() + "/p2p-circuit/p2p/" + client.ID().Pretty()

	log.Println("go-wrapper, client.go, CreateClientHost, My Peer ID: ", client.ID().Pretty())
	log.Println("go-wrapper, client.go, CreateClientHost, My Address: ", fullAddr)

	return client
}

// NewClient sets up a new Client with configuration `cfg`.
// The Client:
//  - imports the keystore and unlocks the account
//  - listens on IP:port
//  - connects to the eth node
//  - in case either the Adjudicator and AssetHolder of the `cfg` are nil, it
//    deploys needed contract. There is currently no check that the
//    correct bytecode is deployed to the given addresses if they are
//    not nil.
//  - sets the `cfg`s Adjudicator and AssetHolder to the deployed contracts
//    addresses in case they were deployed.
func NewClient(ctx *Context, cfg *Config, w *Wallet, secretKey string) (*Client, error) {
	log.Println("go-wrapper, client.go, NewClient, 1")

	// Verbinde mit Relay
	host := CreateClientHost(secretKey)

	log.Println("go-wrapper, client.go, NewClient, 1.5")

	endpoint := fmt.Sprintf("%s:%d", cfg.IP, cfg.Port)
	//listener, err := simple.NewTCPListener(endpoint)
	listener, err := NewTCPListenerP2P(host)
	if err != nil {
		return nil, errors.WithMessagef(err, "listening on %s", endpoint)
	}
	log.Println("go-wrapper, client.go, NewClient, 2")
	//dialer := simple.NewTCPDialer(time.Second * 15)
	dialer := NewTCPDialerP2P(time.Second*15, host)
	ethClient, err := ethclient.Dial(cfg.ETHNodeURL)
	if err != nil {
		return nil, errors.WithMessage(err, "connecting to ethereum node")
	}
	log.Println("go-wrapper, client.go, NewClient, 3")

	acc, err := w.unlock(*cfg.Address)
	if err != nil {
		return nil, errors.WithMessage(err, "finding account")
	}

	signer := types.NewEIP155Signer(big.NewInt(1337))
	cb := ethchannel.NewContractBackend(ethClient, keystore.NewTransactor(*w.w, signer))
	if err := setupContracts(ctx.ctx, cb, acc.Account, cfg); err != nil {
		return nil, errors.WithMessage(err, "setting up contracts")
	}
	log.Println("go-wrapper, client.go, NewClient,  4")

	bus := net.NewBus(acc, dialer)
	adjudicator := ethchannel.NewAdjudicator(cb, common.Address(cfg.Adjudicator.addr), acc.Account.Address, acc.Account)
	accs := map[ethchannel.Asset]accounts.Account{cfg.AssetHolder.addr: acc.Account}
	depositor := new(ethchannel.ETHDepositor)
	deps := map[ethchannel.Asset]ethchannel.Depositor{cfg.AssetHolder.addr: depositor}
	log.Println("go-wrapper, client.go, NewClient, 5")

	funder := ethchannel.NewFunder(cb, accs, deps)
	c, err := client.New(acc.Address(), bus, funder, adjudicator, w.w)
	if err != nil {
		return nil, errors.WithMessage(err, "creating client")
	}
	go bus.Listen(listener)
	log.Println("go-wrapper, client.go, NewClient, 6")

	return &Client{cfg: cfg, ethClient: ethClient,
		client:    c,
		persister: nil,
		wallet:    w.w,
		onChain:   acc,
		dialer:    dialer,
		Bus:       bus,
		PeerID:    host.ID().Pretty()}, nil
}

// Close closes the client and its PersistRestorer to synchronize the database.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Close
// ref https://pkg.go.dev/perun.network/go-perun/channel/persistence/keyvalue?tab=doc#PersistRestorer.Close
func (c *Client) Close() error {
	log.Println("go-wrapper, client.go, Close, Beginn")
	if err := c.client.Close(); err != nil {
		return errors.WithMessage(err, "closing client")
	}
	if err := c.Bus.Close(); err != nil {
		return errors.WithMessage(err, "closing bus")
	}
	if c.persister != nil {
		return errors.WithMessage(c.persister.Close(), "closing persister")
	}
	log.Println("go-wrapper, client.go, Close, Ende")
	return nil
}

// Handle is the handler routine for channel proposals and channel updates. It
// must only be started at most once by the user.
// Incoming proposals and updates are forwarded to the passed handlers.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.Handle
func (c *Client) Handle(ph ProposalHandler, uh UpdateHandler) {
	log.Println("go-wrapper, client.go, Handle, Beginn")
	c.client.Handle(&proposalHandler{c: c, h: ph}, &updateHandler{h: uh})
	log.Println("go-wrapper, client.go, Handle, Ende")
}

// OnNewChannel sets a handler to be called whenever a new channel is created
// or restored. Only one such handler can be set at a time, and repeated calls
// to this function will overwrite the currently existing handler. This
// function may be safely called at any time.
// Start the watcher routine here, if needed.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.OnNewChannel
func (c *Client) OnNewChannel(callback NewChannelCallback) {
	log.Println("go-wrapper, client.go, OnNewChannel, Beginn")
	c.client.OnNewChannel(func(ch *client.Channel) {
		callback.OnNew(&PaymentChannel{ch})
	})
	log.Println("go-wrapper, client.go, OnNewChannel, Ende")
}

// EnablePersistence loads or creates a levelDB database at the given `dbPath`
// and tries to restore all channels from it.
// After this function was successfully called, all changes to the Client are
// saved to the database.
// This function is not thread safe.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.EnablePersistence
func (c *Client) EnablePersistence(dbPath string) (err error) {
	log.Println("go-wrapper, client.go, EnablePersistence, 1")
	var db *leveldb.Database

	log.Println("go-wrapper, client.go, dbpath: ", dbPath)
	db, err = leveldb.LoadDatabase(dbPath)
	log.Println("go-wrapper, client.go, EnablePersistence, 2")
	if err != nil {
		return errors.WithMessage(err, "creating/loading database")
	}
	log.Println("go-wrapper, client.go, EnablePersistence, 3")
	c.persister = keyvalue.NewPersistRestorer(db)
	log.Println("go-wrapper, client.go, EnablePersistence, 4")
	c.client.EnablePersistence(c.persister)
	log.Println("go-wrapper, client.go, EnablePersistence, 5")
	return nil
}

// Restore restores all channels from persistence. Channels are restored in
// parallel. Newly restored channels should be acquired through the
// OnNewChannel callback.
// Note that connections are currently established serially, so allow for
// enough time in the passed context.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.Restore
func (c *Client) Restore(ctx *Context) error {
	log.Println("go-wrapper, client.go, Restore, 1")
	if c.persister == nil {
		return errors.New("persistence not enabled")
	}
	return c.client.Restore(ctx.ctx)
}

// AddPeer adds a new peer to the client. Must be called before proposing
// a new channel with said peer. Wraps go-perun/peer/net/Dialer.Register.
// ref https://pkg.go.dev/perun.network/go-perun/peer/net?tab=doc#Dialer.Register
func (c *Client) AddPeer(perunID *Address, peerID string) {
	log.Println("go-wrapper, client.go, AddPeer, 1")
	c.dialer.Register((*ethwallet.Address)(&perunID.addr), peerID) // instead of fmt.Sprintf("%s:%d", host, port)
}

// setupContracts checks which contracts of the `cfg` are nil and deploys them
// to the blockchain. Writes the addresses of the deployed contracts back to
// the `cfg` struct.
func setupContracts(ctx context.Context, cb ethchannel.ContractBackend, deployer accounts.Account, cfg *Config) error {
	log.Println("go-wrapper, client.go, setupContracts, 1")
	if cfg.Adjudicator == nil {
		adjudicator, err := ethchannel.DeployAdjudicator(ctx, cb, deployer)
		if err != nil {
			return errors.WithMessage(err, "deploying adjudicator")
		}
		cfg.Adjudicator = &Address{ethwallet.Address(adjudicator)}
	}
	if cfg.AssetHolder == nil {
		assetHolder, err := ethchannel.DeployETHAssetholder(ctx, cb, common.Address(cfg.Adjudicator.addr), deployer)
		if err != nil {
			return errors.WithMessage(err, "deploying eth assetHolder")
		}
		cfg.AssetHolder = &Address{ethwallet.Address(assetHolder)}
	}
	// The deployment itself is already logged in the `DeployX` methods
	log.WithFields(log.Fields{"adjudicator": cfg.Adjudicator.ToHex(), "assetHolder": cfg.AssetHolder.ToHex()}).Debugf("Set contracts")
	return nil
}

// OnChainBalance returns the on-chain balance for `address` in Wei.
func (c *Client) OnChainBalance(ctx *Context, address *Address) (*BigInt, error) {
	log.Println("go-wrapper, client.go, OnChainBalance, 1")
	bal, err := c.ethClient.BalanceAt(ctx.ctx, common.Address(address.addr), nil)
	return &BigInt{bal}, err
}
