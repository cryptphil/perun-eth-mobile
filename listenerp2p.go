// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"io"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"

	"perun.network/go-perun/log"
	wirenet "perun.network/go-perun/wire/net"
)

// ListenerP2P implements the go-perun/wire/net/Listener interface
type ListenerP2P struct {
	myHost host.Host
	myRwc  io.ReadWriteCloser
}

// NewTCPListenerP2P ...
func NewTCPListenerP2P(host host.Host) (*ListenerP2P, error) {
	log.Println("go-wrapper, listenerp2p.go, NewTCPListenerP2P, 1")
	//var myListener ListenerP2P = ListenerP2P{myHost: host, myRwc: nil}
	myListener := ListenerP2P{myHost: host, myRwc: nil}
	host.SetStreamHandler("/client", func(s network.Stream) {
		log.Println("\nGot a new Stream!")
		//rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		//reader := bufio.NewReader(s)
		//writer := bufio.NewWriter(s)
		//var rwc io.ReadWriteCloser = &ClosableBufio{*reader, *writer}
		myListener.myRwc = s
		//myListener.Accept)
	})

	log.Println("go-wrapper, listenerp2p.go, NewTCPListenerP2P, 2")
	return &myListener, nil
}

// Accept ..
func (l *ListenerP2P) Accept() (wirenet.Conn, error) {
	log.Println("go-wrapper, listenerp2p.go, Accept, 1")
	var tmp io.ReadWriteCloser
	for {
		if l.myRwc != nil {
			tmp = l.myRwc
			l.myRwc = nil
			break
		}
	}
	return wirenet.NewIoConn(tmp), nil
}

// Close ..
func (l *ListenerP2P) Close() error {
	log.Println("go-wrapper, listenerp2p.go, Close, 1")
	return nil
}
