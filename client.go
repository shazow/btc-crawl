package main

import (
	"github.com/btcsuite/btcd/wire"
)

type Client struct {
	btcnet    wire.BitcoinNet // Bitcoin Network
	pver      uint32          // Protocl Version
	userAgent string          // User Agent
}

func NewClient(userAgent string) *Client {
	return &Client{
		btcnet:    wire.MainNet,
		pver:      wire.ProtocolVersion,
		userAgent: userAgent,
	}
}
