package main

import (
	"net"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
)

func GetSeedsFromDNS(dnsSeeds []string) []string {
	wait := sync.WaitGroup{}
	results := make(chan []net.IP)

	for _, seed := range dnsSeeds {
		wait.Add(1)
		go func(address string) {
			defer wait.Done()
			ips, err := net.LookupIP(address)
			if err != nil {
				logger.Warningf("Failed to resolve %s: %v", address, err)
				return
			}
			logger.Debugf("Resolved %d seeds from %s.", len(ips), address)
			results <- ips
		}(seed)
	}

	go func() {
		wait.Wait()
		close(results)
	}()

	seeds := []string{}
	for ips := range results {
		for _, ip := range ips {
			seeds = append(seeds, net.JoinHostPort(ip.String(), chaincfg.MainNetParams.DefaultPort))
		}
	}

	logger.Infof("Resolved %d seed nodes from %d DNS seeds.", len(seeds), len(dnsSeeds))

	// Note that this will likely include duplicate seeds. The crawler deduplicates them.
	return seeds
}
