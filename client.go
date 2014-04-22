package main

import (
	"github.com/conformal/btcwire"
)

// TODO: Unhardcode these:
var userAgent string = "/btc-crawl:0.0.1"
var lastBlock int32 = 0

type Client struct {
	btcnet    btcwire.BitcoinNet // Bitcoin Network
	pver      uint32             // Protocl Version
	userAgent string             // User Agent
	lastBlock int32
}

func NewDefaultClient() *Client {
	return &Client{
		btcnet:    btcwire.MainNet,
		pver:      btcwire.ProtocolVersion,
		userAgent: userAgent,
		lastBlock: lastBlock,
	}
}
