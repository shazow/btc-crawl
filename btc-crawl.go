package main

import (
	"github.com/jessevdk/go-flags"
	"log"
)

// Taken from: https://github.com/bitcoin/bitcoin/blob/89d72f3d9b6e7ef051ad1439f266b809f348229b/src/chainparams.cpp#L143
var defaultDnsSeeds = []string{
	"seed.bitcoin.sipa.be",
	"dnsseed.bluematt.me",
	"dnsseed.bitcoin.dashjr.org",
	"seed.bitcoinstats.com",
	"seed.bitnodes.io",
	"bitseed.xf2.org",
}

// TODO: Unhardcode these:
var userAgent string = "/btc-crawl:0.0.1"
var lastBlock int32 = 0

type Options struct {
	Verbose []bool   `short:"v" long:"verbose" description:"Show verbose logging."`
	Seed    []string `short:"s" long:"seed" description:"Override which seeds to use." default-mask:"<bitcoind DNS seeds>"`
}

func main() {
	// TODO: Parse args.
	options := Options{}
	_, err := flags.Parse(&options)
	if err != nil {
		log.Fatal(err)
	}

	seedNodes := options.Seed

	// TODO: Export to a reasonable format.
	// TODO: Use proper logger for logging.
	if len(seedNodes) == 0 {
		seedNodes = GetSeedsFromDNS(defaultDnsSeeds)
	}

	client := NewClient(userAgent, lastBlock)
	crawler := NewCrawler(client, seedNodes, 10)
	crawler.Start()
}
