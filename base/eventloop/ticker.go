package eventloop

import (
	"context"
	"time"
)

type Ticker struct {
	loop   *EventLoop
	ticker *time.Ticker
	f      func()
	cancel context.CancelFunc
}

func newTicker(loop *EventLoop, d time.Duration, f func()) *Ticker {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := &Ticker{
		loop:   loop,
		ticker: time.NewTicker(d),
		f:      f,
		cancel: cancel,
	}
	go ticker.receiveTime(ctx, ticker.ticker)
	return ticker
}

func (this *Ticker) receiveTime(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			this.loop.RunInLoop(this.f)
		case <-ctx.Done():
			return
		}
	}
}

func (this *Ticker) Stop() {
	this.ticker.Stop()
	this.cancel()
}
