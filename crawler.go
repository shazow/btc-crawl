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
	queue      []string
	peerAge    time.Duration
}

type Result struct {
	Node  *Peer
	Peers []*btcwire.NetAddress
}

func NewCrawler(client *Client, queue []string, peerAge time.Duration) *Crawler {
	c := Crawler{
		client:     client,
		count:      0,
		seenFilter: map[string]bool{},
		queue:      []string{},
		peerAge:    peerAge,
	}

	// Prefill the queue
	for _, address := range queue {
		c.addAddress(address)
	}

	return &c
}

func (c *Crawler) handleAddress(address string) *Result {
	client := c.client
	peer := NewPeer(client, address)
	r := Result{Node: peer}

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
			r.Peers = append(r.Peers, tmsg.AddrList...)

			if firstReceived == -1 {
				firstReceived = len(tmsg.AddrList)
			} else if firstReceived > len(tmsg.AddrList) || firstReceived == 0 {
				// Probably done.
				return &r
			}
		default:
			otherMessages = append(otherMessages, tmsg.Command())
			if len(otherMessages) > tolerateMessages {
				logger.Debugf("[%s] Giving up with %d results after tolerating messages: %v.", address, len(r.Peers), otherMessages)
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

func (c *Crawler) Run(numWorkers int, stopAfter int) *[]Result {
	numActive := 0
	numGood := 0

	resultChan := make(chan Result)
	workerChan := make(chan struct{}, numWorkers)

	results := []Result{}

	if stopAfter == 0 {
		// No stopping.
		stopAfter = -1
	}

	// This is the main "event loop". Feels like there may be a better way to
	// manage the number of concurrent workers but I can't think of it right now.
	for {
		select {
		case workerChan <- struct{}{}:
			if len(c.queue) == 0 {
				// No work yet.
				<-workerChan
				continue
			}

			// Pop from the queue
			address := c.queue[0]
			c.queue = c.queue[1:]
			numActive += 1

			go func() {
				logger.Debugf("[%s] Worker started.", address)
				resultChan <- *c.handleAddress(address)
			}()

		case r := <-resultChan:
			newAdded := 0
			timestampSince := time.Now().Add(-c.peerAge)

			for _, addr := range r.Peers {
				if !addr.Timestamp.After(timestampSince) {
					continue
				}

				if c.addAddress(NetAddressKey(addr)) {
					newAdded += 1
				}
			}

			if newAdded > 0 {
				numGood += 1
			}
			numActive -= 1

			if len(r.Peers) > 0 {
				stopAfter--
				results = append(results, r)

				logger.Infof("Added %d new peers of %d returned. Total %d known peers via %d connected.", newAdded, len(r.Peers), c.count, numGood)
			}

			if stopAfter == 0 || (len(c.queue) == 0 && numActive == 0) {
				logger.Infof("Done.")
				return &results
			}

			<-workerChan
		}
	}
}
