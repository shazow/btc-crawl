package main

import (
	"time"

	"./queue"

	"github.com/conformal/btcwire"
)

// TODO: Break Client/Peer/Crawler into separate modules.
type Crawler struct {
	client       *Client
	queue        *queue.Queue
	numSeen      int
	numUnique    int
	numConnected int
	numAttempted int
	seenFilter   map[string]bool // TODO: Replace with bloom filter?
	peerAge      time.Duration
}

type Result struct {
	Node  *Peer
	Peers []*btcwire.NetAddress
}

func NewCrawler(client *Client, seeds []string, peerAge time.Duration) *Crawler {
	c := Crawler{
		client:     client,
		seenFilter: map[string]bool{},
		peerAge:    peerAge,
	}
	filter := func(address string) *string {
		return c.filter(address)
	}
	c.queue = queue.NewQueue(filter, 10)

	go func() {
		// Prefill the queue
		for _, address := range seeds {
			c.queue.Input <- address
		}
	}()

	return &c
}

func (c *Crawler) handleAddress(address string) *Result {
	c.numAttempted++

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

	c.numConnected++

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

func (c *Crawler) filter(address string) *string {
	// Returns true if not seen before, otherwise false
	c.numSeen++

	state, ok := c.seenFilter[address]
	if ok == true && state == true {
		return nil
	}

	c.seenFilter[address] = true
	c.numUnique++
	return &address
}

/*
func (c *Crawler) Run(resultChan chan<- Result, numWorkers int) {
	workChan := make(chan string, numWorkers)
	queueChan := make(chan string)
	tempResult := make(chan Result)

	go func(queueChan <-chan string) {
		// Single thread to safely manage the queue
		c.addAddress(<-queueChan)
		nextAddress, _ := c.popAddress()

		for {
			select {
			case address := <-queueChan:
				// Enque address
				c.addAddress(address)
			case workChan <- nextAddress:
				nextAddress, err := c.popAddress()
				if err != nil {
					// Block until we get more work
					c.addAddress(<-queueChan)
					nextAddress, _ = c.popAddress()
				}
			}
		}
	}(queueChan)

	go func(tempResult <-chan Result, workChan chan<- string) {
		// Convert from result to queue.
		for {
			select {
			case r := <-tempResult:

			}
		}
	}(tempResult, workChan)

	for address := range workChan {
		// Spawn more workers as we get buffered work
		go func() {
			logger.Debugf("[%s] Worker started.", address)
			tempResult <- *c.handleAddress(address)
		}()
	}
}
*/

func (c *Crawler) Run(numWorkers int, stopAfter int) *[]Result {
	numActive := 0

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
			go func() {
				address := <-c.queue.Output
				logger.Debugf("[%s] Worker started.", address)
				resultChan <- *c.handleAddress(address)
			}()

		case r := <-resultChan:
			timestampSince := time.Now().Add(-c.peerAge)

			for _, addr := range r.Peers {
				if !addr.Timestamp.After(timestampSince) {
					continue
				}

				c.queue.Input <- NetAddressKey(addr)
			}

			numActive--

			if len(r.Peers) > 0 {
				stopAfter--
				results = append(results, r)

				logger.Infof("[%s] Returned %d peers. Total %d unique peers via %d connected (of %d attempted).", r.Node.Address, len(r.Peers), c.numUnique, c.numConnected, c.numAttempted)
			}

			if stopAfter == 0 || (c.queue.IsEmpty() && numActive == 0) {
				logger.Infof("Done.")
				return &results
			}

			<-workerChan
		}
	}
}
