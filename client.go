package main

import (
	"github.com/conformal/btcwire"
)

type Client struct {
	btcnet    btcwire.BitcoinNet // Bitcoin Network
	pver      uint32             // Protocl Version
	userAgent string             // User Agent
	lastBlock int32
}

func NewClient(userAgent string, lastBlock int32) *Client {
	return &Client{
		btcnet:    btcwire.MainNet,
		pver:      btcwire.ProtocolVersion,
		userAgent: userAgent,
		lastBlock: lastBlock,
	}
}
