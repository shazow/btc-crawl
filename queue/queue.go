package queue

// A single goroutine manages the overflow queue for thread-safety, funneling
// data between the Input and Output channels through a specified filter.
type Queue struct {
	Input    chan string
	Output   chan string
	overflow []string
	filter   func(string) *string
	count    int
}

func NewQueue(filter func(string) *string, bufferSize int) *Queue {
	q := Queue{
		Input:    make(chan string, bufferSize),
		Output:   make(chan string, bufferSize),
		overflow: []string{},
		filter:   filter,
	}

	go func(input <-chan string, output chan<- string) {
		// Block until we have a next item
		nextItem := q.next()

		for {
			select {
			case item := <-input:
				// New input
				r := q.filter(item)
				if r != nil {
					// Store in the overflow
					q.overflow = append(q.overflow, *r)
					q.count++
				}
			case output <- nextItem:
				// Block until we have more inputs
				nextItem = q.next()
			}
		}
	}(q.Input, q.Output)

	return &q
}

func (q *Queue) next() string {
	// Block until a next item is available.

	if len(q.overflow) > 0 {
		// Pop off the overflow queue.
		r := q.overflow[0]
		q.overflow = q.overflow[1:]
		return r
	}

	for {
		// Block until we have a viable output
		r := q.filter(<-q.Input)

		if r != nil {
			q.count++
			return *r
		}
	}
}

func (q *Queue) IsEmpty() bool {
	// FIXME: This effectively cycles the order of the output buffer. Kinda sad.

	if len(q.overflow) > 0 {
		return false
	}

	select {
	case r := <-q.Output:
		go func() {
			// Put it back
			q.Output <- r
		}()
		return false
	default:
		return true
	}
}

func (q *Queue) Wait() {
	// Wait until there is an Output. Useful for blocking until the queue is
	// ramped up.
	r := <-q.Output
	go func() {
		q.Output <- r
	}()
}

func (q *Queue) Count() int {
	// Number of outputs produced.
	return q.count
}
