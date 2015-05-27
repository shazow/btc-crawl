package main

import (
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"
)

// TODO: Break Client/Peer/Crawler into separate modules.
type Crawler struct {
	client         *Client
	queue          *Queue
	numSeen        int
	numUnique      int
	numConnected   int
	numAttempted   int
	seenFilter     map[string]bool // TODO: Replace with bloom filter?
	PeerAge        time.Duration
	ConnectTimeout time.Duration
	shutdown       chan struct{}
	waitGroup      sync.WaitGroup
}

type Result struct {
	Node  *Peer
	Peers []*wire.NetAddress
}

func NewCrawler(client *Client, seeds []string) *Crawler {
	c := Crawler{
		client:     client,
		seenFilter: map[string]bool{},
		shutdown:   make(chan struct{}, 1),
		waitGroup:  sync.WaitGroup{},
	}
	filter := func(address string) *string {
		return c.filter(address)
	}

	done := make(chan struct{})
	c.queue = NewQueue(filter, done)

	// Prefill the queue
	for _, address := range seeds {
		c.addSeed(address)
	}

	go func() {
		c.waitGroup.Wait()
		done <- struct{}{}
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
	err = peer.WriteMessage(wire.NewMsgGetAddr())
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
		if len(otherMessages) > tolerateMessages {
			logger.Debugf("[%s] Giving up with %d results after tolerating messages: %v.", address, len(r.Peers), otherMessages)
			return &r
		}

		msg, _, err := peer.ReadMessage()
		if err != nil {
			otherMessages = append(otherMessages, err.Error())
			logger.Warningf("[%s] Failed to read message: %v", address, err)
			continue
		}

		switch tmsg := msg.(type) {
		case *wire.MsgAddr:
			r.Peers = append(r.Peers, tmsg.AddrList...)

			if firstReceived == -1 {
				firstReceived = len(tmsg.AddrList)
			} else if firstReceived > len(tmsg.AddrList) || firstReceived == 0 {
				// Probably done.
				return &r
			}
		default:
			otherMessages = append(otherMessages, tmsg.Command())
		}
	}
}

func (c *Crawler) addSeed(address string) {
	c.waitGroup.Add(1)
	if c.queue.Add(address) == false {
		c.waitGroup.Done()
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
	numStalePeers := 0

	for _, addr := range r.Peers {
		if !addr.Timestamp.After(timestampSince) {
			numStalePeers++
			continue
		}

		c.addSeed(NetAddressKey(addr))
	}

	if len(r.Peers) > 0 {
		logger.Infof("[%s] Returned %d peers (including %d stale). Total %d unique peers via %d connected (of %d attempted).", r.Node.Address, len(r.Peers), numStalePeers, c.numUnique, c.numConnected, c.numAttempted)
		return r
	}

	return nil
}

func (c *Crawler) Run(numWorkers int) <-chan Result {
	result := make(chan Result, 100)
	workerChan := make(chan struct{}, numWorkers)
	isDone := false

	go func() {
		// Queue handler
		for address := range c.queue.Iter() {
			// Reserve worker slot (block)
			workerChan <- struct{}{}

			if isDone {
				break
			}

			// Start worker
			go func() {
				r := c.handleAddress(address)

				// Process the result
				if c.process(r) != nil {
					result <- *r
				}

				// Clear worker slot
				<-workerChan
				c.waitGroup.Done()
			}()
		}

		logger.Infof("Stopping queue after %d added items.", c.queue.Count())
		close(result)
	}()

	go func() {
		<-c.shutdown
		isDone = true
	}()

	return result
}
