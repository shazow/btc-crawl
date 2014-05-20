package main

import "sync"

// TODO: Make this an interface and multiple implementations (Redis etc?)
type Queue struct {
	sync.Mutex
	storage []string
	filter  func(string) *string
	count   int
	cond    *sync.Cond
	done    <-chan struct{}
}

func NewQueue(filter func(string) *string, done <-chan struct{}) *Queue {
	q := Queue{
		storage: []string{},
		filter:  filter,
		done:    done,
	}
	q.cond = sync.NewCond(&q)

	return &q
}

func (q *Queue) Add(item string) bool {
	q.Lock()
	r := q.filter(item)
	if r == nil {
		q.Unlock()
		return false
	}

	q.storage = append(q.storage, *r)
	q.count++
	q.Unlock()
	q.cond.Signal()

	return true
}

func (q *Queue) Iter() <-chan string {
	ch := make(chan string)

	go func() {
		<-q.done
		q.cond.Signal() // Wake up to close the channel.
	}()

	go func() {
		for {
			q.Lock()
			if len(q.storage) == 0 {
				// Wait until next Add
				q.cond.Wait()

				if len(q.storage) == 0 {
					// Queue is finished
					close(ch)
					q.Unlock()
					return
				}
			}

			r := q.storage[0]
			q.storage = q.storage[1:]
			q.Unlock()

			ch <- r
		}
	}()

	return ch
}

func (q *Queue) Count() int {
	// Number of outputs produced.
	return q.count
}
