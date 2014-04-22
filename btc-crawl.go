package main

import (
	"log"
)

func main() {
	// TODO: Parse args.
	// TODO: Export to a reasonable format.
	// TODO: Use proper logger for logging.
	var seedNodes []string = []string{"85.214.251.25:8333", "62.75.216.13:8333"}
	client := NewDefaultClient()
	crawler := NewCrawler(client, seedNodes, 10)

	done, err := crawler.Start()
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
