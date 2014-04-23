package main

import (
	"github.com/conformal/btcwire"
)

type Client struct {
	btcnet    btcwire.BitcoinNet // Bitcoin Network
	pver      uint32             // Protocl Version
	userAgent string             // User Agent
}

func NewClient(userAgent string) *Client {
	return &Client{
		btcnet:    btcwire.MainNet,
		pver:      btcwire.ProtocolVersion,
		userAgent: userAgent,
	}
}
