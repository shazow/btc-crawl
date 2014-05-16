package main

import (
	"time"

	"./queue"

	"github.com/conformal/btcwire"
)

// TODO: Break Client/Peer/Crawler into separate modules.
type Crawler struct {
	client         *Client
	queue          *queue.Queue
	numSeen        int
	numUnique      int
	numConnected   int
	numAttempted   int
	seenFilter     map[string]bool // TODO: Replace with bloom filter?
	PeerAge        time.Duration
	ConnectTimeout time.Duration
	shutdown       chan struct{}
}

type Result struct {
	Node  *Peer
	Peers []*btcwire.NetAddress
}

func NewCrawler(client *Client, seeds []string) *Crawler {
	c := Crawler{
		client:     client,
		seenFilter: map[string]bool{},
		shutdown:   make(chan struct{}, 1),
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

func (c *Crawler) Shutdown() {
	c.shutdown <- struct{}{}
}

func (c *Crawler) handleAddress(address string) *Result {
	c.numAttempted++

	client := c.client
	peer := NewPeer(client, address)
	peer.ConnectTimeout = c.ConnectTimeout
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

func (c *Crawler) process(r *Result) *Result {
	timestampSince := time.Now().Add(-c.PeerAge)

	for _, addr := range r.Peers {
		if !addr.Timestamp.After(timestampSince) {
			continue
		}

		c.queue.Input <- NetAddressKey(addr)
	}

	if len(r.Peers) > 0 {
		logger.Infof("[%s] Returned %d peers. Total %d unique peers via %d connected (of %d attempted).", r.Node.Address, len(r.Peers), c.numUnique, c.numConnected, c.numAttempted)
		return r
	}

	return nil
}

func (c *Crawler) Run(resultChan chan<- Result, numWorkers int) {
	workerChan := make(chan struct{}, numWorkers)
	tempResult := make(chan Result)
	numActive := 0
	isActive := true

	// Block until we get the first item
	c.queue.Wait()

	// This is the main "event loop".
	// FIXME: Feels like there should be a better way to manage the number of
	// concurrent workers without limiting slots with workerChan and without
	// using a numActive counter.
	for {
		select {
		case workerChan <- struct{}{}:
			if !isActive {
				// Don't start any new workers, leave the slot filled.
				break
			} else if c.queue.IsEmpty() {
				<-workerChan

				if numActive == 0 {
					logger.Infof("Done.")
					close(resultChan)
					return
				}
			}

			numActive++
			go func() {
				address := <-c.queue.Output
				logger.Debugf("[%s] Worker started.", address)
				tempResult <- *c.handleAddress(address)
			}()

		case r := <-tempResult:
			if c.process(&r) != nil {
				resultChan <- r
			}
			numActive--
			<-workerChan

		case <-c.shutdown:
			logger.Infof("Shutting down.")
			isActive = false
		}
	}
}
