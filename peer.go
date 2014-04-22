package main

import (
	"fmt"
	"github.com/conformal/btcwire"
	"log"
	"net"
)

type Peer struct {
	client  *Client
	address string
	conn    net.Conn
	nonce   uint64 // Nonce we're sending to the peer
}

func NewPeer(client *Client, address string) *Peer {
	p := Peer{
		client:  client,
		address: address,
	}
	return &p
}

func (p *Peer) Connect() error {
	if p.conn != nil {
		return fmt.Errorf("Peer already connected, can't connect again.")
	}
	conn, err := net.Dial("tcp", p.address)
	if err != nil {
		return err
	}

	p.conn = conn
	return nil
}

func (p *Peer) Disconnect() {
	p.conn.Close()
}

func (p *Peer) Handshake() error {
	if p.conn == nil {
		return fmt.Errorf("Peer is not connected, can't handshake.")
	}

	log.Printf("[%s] Starting handshake.", p.address)

	nonce, err := btcwire.RandomUint64()
	if err != nil {
		return err
	}
	p.nonce = nonce

	pver, btcnet := p.client.pver, p.client.btcnet

	msgVersion, err := btcwire.NewMsgVersionFromConn(p.conn, p.nonce, p.client.userAgent, 0)
	msgVersion.DisableRelayTx = true
	if err := btcwire.WriteMessage(p.conn, msgVersion, pver, btcnet); err != nil {
		return err
	}

	// Read the response version.
	msg, _, err := btcwire.ReadMessage(p.conn, pver, btcnet)
	if err != nil {
		return err
	}
	vmsg, ok := msg.(*btcwire.MsgVersion)
	if !ok {
		return fmt.Errorf("Did not receive version message: %T", vmsg)
	}
	// Negotiate protocol version.
	if uint32(vmsg.ProtocolVersion) < pver {
		pver = uint32(vmsg.ProtocolVersion)
	}
	log.Printf("[%s] -> Version: %s", p.address, vmsg.UserAgent)

	// Normally we'd check if vmsg.Nonce == p.nonce but the crawler does not
	// accept external connections so we skip it.

	// Send verack.
	if err := btcwire.WriteMessage(p.conn, btcwire.NewMsgVerAck(), pver, btcnet); err != nil {
		return err
	}

	return nil
}
