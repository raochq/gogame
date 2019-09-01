package eventloop

import (
	"context"
	"time"
)

type Timer struct {
	loop   *EventLoop
	timer  *time.Timer
	f      func()
	cancel context.CancelFunc
}

func newTimer(loop *EventLoop, d time.Duration, f func()) *Timer {
	ctx, cancel := context.WithCancel(context.Background())
	timer := &Timer{
		loop:   loop,
		timer:  time.NewTimer(d),
		f:      f,
		cancel: cancel,
	}
	go timer.receiveTime(ctx, timer.timer)
	return timer
}

func (this *Timer) receiveTime(ctx context.Context, timer *time.Timer) {
	select {
	case <-timer.C:
		this.loop.RunInLoop(this.f)
	case <-ctx.Done():
	}
}

func (this *Timer) Stop() bool {
	if this.timer.Stop() {
		this.cancel()
		return true
	}
	return false
}
