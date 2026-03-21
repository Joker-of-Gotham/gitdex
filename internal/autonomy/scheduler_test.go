package autonomy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduler_RegisterAndStart(t *testing.T) {
	sched := NewScheduler()
	var count int32
	sched.Register(SchedulerTask{
		Name:     "test",
		Interval: 10 * time.Millisecond,
		Action: func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)
	time.Sleep(35 * time.Millisecond) // ~3 ticks
	sched.Stop()
	cancel()

	if c := atomic.LoadInt32(&count); c < 2 {
		t.Errorf("expected at least 2 runs, got %d", c)
	}
}

func TestScheduler_StopIdempotent(t *testing.T) {
	sched := NewScheduler()
	sched.Register(SchedulerTask{
		Name:     "noop",
		Interval: time.Hour,
		Action:   func(ctx context.Context) error { return nil },
	})
	ctx := context.Background()
	sched.Start(ctx)

	sched.Stop()
	sched.Stop() // should not panic
}
