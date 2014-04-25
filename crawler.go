package main

import (
	"time"

	"github.com/conformal/btcwire"
)

// TODO: Break Client/Peer/Crawler into separate modules.
type Crawler struct {
	client     *Client
	count      int
	seenFilter map[string]bool // TODO: Replace with bloom filter?
	results    chan []string
	workers    chan struct{}
	queue      []string
	peerAge    time.Duration
}

func NewCrawler(client *Client, queue []string, numWorkers int, peerAge time.Duration) *Crawler {
	c := Crawler{
		client:     client,
		count:      0,
		seenFilter: map[string]bool{},
		results:    make(chan []string),
		workers:    make(chan struct{}, numWorkers),
		queue:      []string{},
		peerAge:    peerAge,
	}

	// Prefill the queue
	for _, address := range queue {
		c.addAddress(address)
	}

	return &c
}

func (c *Crawler) handleAddress(address string) *[]string {
	r := []string{}

	client := c.client
	peer := NewPeer(client, address)

	err := peer.Connect()
	if err != nil {
		logger.Debugf("[%s] Connection failed: %v", address, err)
		return &r
	}
	defer peer.Disconnect()

	err = peer.Handshake()
	if err != nil {
		logger.Debugf("[%s] Handsake failed: %v", address, err)
		return &r
	}

	// Send getaddr.
	err = peer.WriteMessage(btcwire.NewMsgGetAddr())
	if err != nil {
		logger.Warningf("[%s] GetAddr failed: %v", address, err)
		return &r
	}

	// Listen for tx inv messages.
	firstReceived := -1
	tolerateMessages := 3
	otherMessages := []string{}
	timestampSince := time.Now().Add(-c.peerAge)

	for {
		// We can't really tell when we're done receiving peers, so we stop either
		// when we get a smaller-than-normal set size or when we've received too
		// many unrelated messages.
		msg, _, err := peer.ReadMessage()
		if err != nil {
			logger.Warningf("[%s] Failed to read message: %v", address, err)
			continue
		}

		switch tmsg := msg.(type) {
		case *btcwire.MsgAddr:
			for _, addr := range tmsg.AddrList {
				if addr.Timestamp.After(timestampSince) {
					// TODO: Move this check to .Start()?
					r = append(r, NetAddressKey(addr))
				}
			}

			if firstReceived == -1 {
				firstReceived = len(tmsg.AddrList)
			} else if firstReceived > len(tmsg.AddrList) || firstReceived == 0 {
				// Probably done.
				return &r
			}
		default:
			otherMessages = append(otherMessages, tmsg.Command())
			if len(otherMessages) > tolerateMessages {
				logger.Debugf("[%s] Giving up with %d results after tolerating messages: %v.", address, len(r), otherMessages)
				return &r
			}
		}
	}
}

func (c *Crawler) addAddress(address string) bool {
	// Returns true if not seen before, otherwise false
	state, ok := c.seenFilter[address]
	if ok == true && state == true {
		return false
	}

	c.seenFilter[address] = true
	c.count += 1
	c.queue = append(c.queue, address)

	return true
}

func (c *Crawler) Start() {
	numWorkers := 0
	numGood := 0

	// This is the main "event loop". Feels like there may be a better way to
	// manage the number of concurrent workers but I can't think of it right now.
	for {
		select {
		case c.workers <- struct{}{}:
			if len(c.queue) == 0 {
				// No work yet.
				<-c.workers
				continue
			}

			// Pop from the queue
			address := c.queue[0]
			c.queue = c.queue[1:]
			numWorkers += 1

			go func() {
				logger.Debugf("[%s] Worker started.", address)
				results := *c.handleAddress(address)
				c.results <- results
			}()

		case r := <-c.results:
			newAdded := 0
			for _, address := range r {
				if c.addAddress(address) {
					newAdded += 1
				}
			}

			if newAdded > 0 {
				numGood += 1
			}
			numWorkers -= 1

			if len(r) > 0 {
				logger.Infof("Added %d new peers of %d returned. Total %d known peers via %d connected.", newAdded, len(r), c.count, numGood)
			}

			if len(c.queue) == 0 && numWorkers == 0 {
				logger.Infof("Done.")
				return
			}

			<-c.workers
		}
	}
}
