package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
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
	Verbose        []bool        `short:"v" long:"verbose" description:"Show verbose logging."`
	Output         string        `short:"o" long:"output" description:"File to write result to." default:"btc-crawl.json"`
	Seed           []string      `short:"s" long:"seed" description:"Override which seeds to use." default-mask:"<bitcoin-core DNS seeds>"`
	Concurrency    int           `short:"c" long:"concurrency" description:"Maximum number of concurrent connections to open." default:"10"`
	ConnectTimeout time.Duration `short:"t" long:"connect-timeout" description:"Maximum time to wait to connect before giving up." default:"10s"`
	UserAgent      string        `short:"A" long:"user-agent" description:"Client name to advertise while crawling. Should be in format of '/name:x.y.z/'." default:"/btc-crawl:0.1.1/"`
	PeerAge        time.Duration `long:"peer-age" description:"Ignore discovered peers older than this." default:"24h"`
	StopAfter      int           `long:"stop-after" description:"Stop crawling after this many results." default:"0"`
}

var logLevels = []log.Level{
	log.Warning,
	log.Info,
	log.Debug,
}

func main() {
	now := time.Now()
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)

	p, err := parser.Parse()
	if err != nil {
		if p == nil {
			fmt.Print(err)
		}
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

	// Create client and crawler
	client := NewClient(options.UserAgent)
	crawler := NewCrawler(client, seedNodes)
	crawler.PeerAge = options.PeerAge
	crawler.ConnectTimeout = options.ConnectTimeout

	// Configure output
	var w *bufio.Writer
	if options.Output == "-" || options.Output == "" {
		w = bufio.NewWriter(os.Stdout)
		defer w.Flush()
	} else {
		fp, err := os.Create(options.Output)
		if err != nil {
			logger.Errorf("Failed to create file: %v", err)
			return
		}

		w = bufio.NewWriter(fp)
		defer w.Flush()
		defer fp.Close()
	}

	// Make the first write, make sure everything is cool
	_, err = w.Write([]byte("["))
	if err != nil {
		logger.Errorf("Failed to write result, aborting immediately: %v", err)
		return
	}

	// Construct interrupt handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig // Wait for ^C signal
		logger.Warningf("Interrupt signal detected, shutting down.")
		crawler.Shutdown()
	}()

	// Launch crawler
	resultChan := crawler.Run(options.Concurrency)
	logger.Infof("Crawler started with %d concurrency limit.", options.Concurrency)

	// Start processing results
	count := 0
	for result := range resultChan {
		b, err := json.Marshal(result)
		if err != nil {
			logger.Warningf("Failed to export JSON, skipping: %v", err)
		}

		if count > 0 {
			b = append([]byte(","), b...)
		}

		_, err = w.Write(b)
		if err != nil {
			logger.Errorf("Failed to write result, aborting gracefully: %v", err)
			crawler.Shutdown()
			break
		}

		count++
		if options.StopAfter > 0 && count > options.StopAfter {
			logger.Infof("StopAfter count reached, shutting down gracefully.")
			crawler.Shutdown()
		}
	}

	w.Write([]byte("]")) // No error checking here because it's too late to care.

	logger.Infof("Written %d results after %s: %s", count, time.Now().Sub(now), options.Output)
}
