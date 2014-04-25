// TODO: Export to a reasonable format.
// TODO: Namespace packages properly (outside of `main`)
package main

import (
	"os"
	"time"

	"github.com/alexcesaro/log"
	"github.com/alexcesaro/log/golog"
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
	Verbose     []bool        `short:"v" long:"verbose" description:"Show verbose logging."`
	Seed        []string      `short:"s" long:"seed" description:"Override which seeds to use." default-mask:"<bitcoin-core DNS seeds>"`
	Concurrency int           `short:"c" long:"concurrency" description:"Maximum number of concurrent connections to open." default:"10"`
	UserAgent   string        `short:"A" long:"user-agent" description:"Client name to advertise while crawling. Should be in format of '/name:x.y.z/'." default:"/btc-crawl:0.1.1/"`
	PeerAge     time.Duration `long:"peer-age" description:"Ignore discovered peers older than this." default:"24h"`
}

var logLevels = []log.Level{
	log.Warning,
	log.Info,
	log.Debug,
}

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		// FIXME: Print on some specific errors? Seems Parse prints in most cases.
		return
	}

	// Figure out the log level
	numVerbose := len(options.Verbose)
	if numVerbose > len(logLevels) { // lol math.Min, you floaty bugger.
		numVerbose = len(logLevels)
	}

	logLevel := logLevels[numVerbose]
	logger = golog.New(os.Stderr, logLevel)

	seedNodes := options.Seed
	if len(seedNodes) == 0 {
		seedNodes = GetSeedsFromDNS(defaultDnsSeeds)
	}

	client := NewClient(options.UserAgent)
	crawler := NewCrawler(client, seedNodes, options.Concurrency, options.PeerAge)
	crawler.Start()
}
