package eventloop

import (
	"sync"
	"time"
)

type EventLoop struct {
	functors []func()

	mutex  sync.Mutex
	cond   *sync.Cond
	closed bool
}

func NewEventLoop() *EventLoop {
	loop := &EventLoop{}
	loop.cond = sync.NewCond(&loop.mutex)
	return loop
}

func (this *EventLoop) Loop() {
	for {
		var functors []func()
		var closed bool
		this.mutex.Lock()
		for !this.closed && len(this.functors) == 0 {
			this.cond.Wait()
		}
		functors, this.functors = this.functors, nil // swap
		closed = this.closed
		this.mutex.Unlock()

		for _, functor := range functors {
			functor()
		}
		if closed {
			return
		}
	}
}

func (this *EventLoop) RunInLoop(functor func()) {
	this.mutex.Lock()
	this.functors = append(this.functors, functor)
	this.mutex.Unlock()

	this.cond.Signal()
}

func (this *EventLoop) RunAfter(d time.Duration, f func()) *Timer {
	return newTimer(this, d, f)
}

func (this *EventLoop) RunEvery(d time.Duration, f func()) *Ticker {
	return newTicker(this, d, f)
}

func (this *EventLoop) Close() {
	this.mutex.Lock()
	if this.closed {
		this.mutex.Unlock()
		return
	}
	this.closed = true
	this.mutex.Unlock()

	this.cond.Signal()
}
