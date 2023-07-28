package bsc

import "sync"

type Subscription struct {
	unsub   chan struct{}
	err     chan error
	unsubed bool
	mu      sync.Mutex
}

func NewSubscription(producer func(<-chan struct{}) error) *Subscription {
	sub := &Subscription{
		unsub: make(chan struct{}),
		err:   make(chan error),
	}

	go func() {
		defer func() {
			close(sub.err)
		}()
		err := producer(sub.unsub)
		sub.mu.Lock()
		defer func() {
			sub.mu.Unlock()
		}()
		if !sub.unsubed {
			if err != nil {
				sub.err <- err
			}
			sub.unsubed = true
		}
	}()

	return sub
}

func (o *Subscription) Unsubscribe() {
	o.mu.Lock()
	if o.unsubed {
		o.mu.Unlock()
		return
	}
	o.unsubed = true
	close(o.unsub)
	o.mu.Unlock()
}

func (o *Subscription) Err() <-chan error {
	return o.err
}
