package bsc

import (
	"sync"
)

// TODO data structure
type WaitQueue struct {
	front  int
	rear   int
	length int
	buf    []interface{}
	cond   *sync.Cond
}

func NewWaitQueue(capacity int) *WaitQueue {
	var pool WaitQueue
	pool.front = 0
	pool.rear = 0
	pool.length = 0
	//pool.buf = make([]*types.Header, capacity, capacity)
	pool.buf = make([]interface{}, capacity, capacity)
	pool.cond = sync.NewCond(&sync.Mutex{})
	return &pool
}

func (p *WaitQueue) First() interface{} {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	if p.length > 0 {
		return p.buf[p.front]
	} else {
		return nil
	}
}

func (p *WaitQueue) Last() interface{} {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	if p.length > 0 {
		return p.buf[p.rear]
	} else {
		return nil
	}
}

func (p *WaitQueue) Enqueue(item interface{}) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.length == cap(p.buf) {
		p.cond.Wait()
	}

	p.buf[p.rear] = item
	p.rear++
	p.length++
	if p.rear == cap(p.buf) {
		p.rear = 0
	}
}

func (p *WaitQueue) Dequeue() interface{} {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	item := p.buf[p.front]
	p.buf[p.front] = nil
	p.front++
	if p.front == cap(p.buf) {
		p.front = 0
	}
	if p.length == cap(p.buf) {
		defer p.cond.Broadcast()
	}
	p.length--
	return item
}

func (p *WaitQueue) Front() interface{} {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.buf[p.front]
}

func (p *WaitQueue) Cap() int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return cap(p.buf)
}

func (p *WaitQueue) Len() int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.length
}
