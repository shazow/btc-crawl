// TODO: Export to a reasonable format.
// TODO: Use proper logger for logging.
package main

import (
	"github.com/jessevdk/go-flags"
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

type Options struct {
	Verbose     []bool   `short:"v" long:"verbose" description:"Show verbose logging."`
	Seed        []string `short:"s" long:"seed" description:"Override which seeds to use." default-mask:"<bitcoin-core DNS seeds>"`
	Concurrency int      `short:"c" long:"concurrency" description:"Maximum number of concurrent connections to open." default:"10"`
	UserAgent   string   `short:"A" long:"user-agent" description:"Client name to advertise while crawling. Should be in format of '/name:x.y.z/'." default:"/btc-crawl:0.1.1/"`
}

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		// FIXME: Print on some specific errors? Seems Parse prints in most cases.
		return
	}

	seedNodes := options.Seed

	if len(seedNodes) == 0 {
		seedNodes = GetSeedsFromDNS(defaultDnsSeeds)
	}

	client := NewClient(options.UserAgent)
	crawler := NewCrawler(client, seedNodes, options.Concurrency)
	crawler.Start()
}
