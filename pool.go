package pool

import (
	"context"
	"errors"
	"io"
	"sync"
)

type Pool struct {
	lock sync.RWMutex
	maxActive int
	maxIdle   int
	new       func() io.Closer
	active    chan int
	pool      chan io.Closer
	closed bool
}

// New create a new pool. Factory function will be called when there is no item
// in the pool and active items are less than maxActive. Max pool size is given
// in maxIdle. maxActive can be less than maxIdle, and maxIdle will be set to
// maxActive in that case.
//
// The pool will not be filled when created, use Fill() to fill the pool.
func New(factory func() io.Closer, maxActive, maxIdle int) (*Pool, error) {
	if maxActive <= 0 {
		return nil, errors.New("max active must be positive")
	}
	if maxIdle < 0 {
		return nil, errors.New("max idle must be non-negative")
	}
	if maxIdle > maxActive {
		maxIdle = maxActive
	}
	return &Pool{
		maxActive: maxActive,
		maxIdle:   maxIdle,
		new:       factory,
		active:    make(chan int, maxActive),
		pool:      make(chan io.Closer, maxIdle),
		closed: false,
	}, nil
}

// Get return an item from the pool. If the active items exceed maxActive, it
// will block until some item finishes or the context is done. If the pool is
// empty, a new item will be created and returned.
func (p *Pool) Get(ctx context.Context) (io.Closer, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.closed {return nil, errors.New("pool is closed")}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.active <- 1:
		return p.takeOrCreate(), nil
	}
}

func (p *Pool) takeOrCreate() (item io.Closer) {
	select {
	case item = <-p.pool:
	default:
		item = p.new()
	}
	return
}

// Put add back item in the pool. If the pool is full, the item will be closed.
func (p *Pool) Put(item io.Closer) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.closed {
		_ = item.Close()
		return
	}
	select {
	case p.pool <- item:
	default:
		item.Close()
	}

	select {
	case <- p.active:
	default:
	}
}

// Release the item without put it back in the pool. The function does not
// close the item.
func (p *Pool) Release() {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.closed {return}
	select {
	case <- p.active:
	default:
	}
}

// Close the pool and all the items in it.
func (p *Pool) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.closed {return}
	p.closed = true

outer:
	for {
		select {
		case item := <- p.pool:
			_ = item.Close()
		default:
			break outer
		}
	}

	close(p.pool)
	close(p.active)
}

// Fill the pool to max size.
func (p *Pool) Fill() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.closed {return}
	for i := len(p.pool); i != p.maxIdle; i++ {
		p.pool <- p.new()
	}
}

// Clear all items in the pool.
func (p *Pool) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.closed {return}

outer:
	for {
		select {
		case item := <- p.pool:
			_ = item.Close()
		default:
			break outer
		}
	}
}

// IdleNum return numbers of idle items in the pool.
func (p *Pool) IdleNum() int {
	return len(p.pool)
}

// Freeze locks the pool so that any other operations will block.
func (p *Pool) Freeze() {
	p.lock.Lock()
}

// Thaw unlocks the pool.
func (p *Pool) Thaw() {
	p.lock.Unlock()
}
