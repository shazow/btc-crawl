package main

import (
	"github.com/conformal/btcwire"
	"log"
	"net"
	"sync"
)

func GetSeedsFromDNS(dnsSeeds []string) []string {
	wait := sync.WaitGroup{}
	results := make(chan []net.IP)

	for _, address := range dnsSeeds {
		wait.Add(1)
		go func(address string) {
			defer wait.Done()
			ips, err := net.LookupIP(address)
			if err != nil {
				log.Printf("Failed to resolve %s: %v", address, err)
				return
			}
			log.Printf("Resolved %d seeds from %s.", len(ips), address)
			results <- ips
		}(address)
	}

	go func() {
		wait.Wait()
		close(results)
	}()

	seeds := []string{}
	for ips := range results {
		for _, ip := range ips {
			seeds = append(seeds, net.JoinHostPort(ip.String(), btcwire.MainPort))
		}
	}

	log.Printf("Resolved %d seed nodes from %d DNS seeds.", len(seeds), len(dnsSeeds))

	// Note that this will likely include duplicate seeds. The crawler deduplicates them.
	return seeds
}
