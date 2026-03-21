package autonomy

import (
	"context"
	"log"
	"sync"
	"time"
)

// SchedulerTask defines a periodic task for the Scheduler.
type SchedulerTask struct {
	Name     string
	Interval time.Duration
	Action   func(ctx context.Context) error
}

// Scheduler runs registered periodic tasks.
type Scheduler struct {
	mu       sync.Mutex
	tasks    []SchedulerTask
	done     chan struct{}
	stopOnce sync.Once
}

// NewScheduler creates a new Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make([]SchedulerTask, 0),
		done:  make(chan struct{}),
	}
}

// Register adds a periodic task to the scheduler.
func (s *Scheduler) Register(task SchedulerTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, task)
}

// Start launches a goroutine for each registered task that runs at the specified interval.
// Each goroutine respects context cancellation and the Stop signal.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	tasks := make([]SchedulerTask, len(s.tasks))
	copy(tasks, s.tasks)
	s.mu.Unlock()

	for _, t := range tasks {
		if t.Interval <= 0 || t.Action == nil {
			continue
		}
		task := t
		go func() {
			ticker := time.NewTicker(task.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-s.done:
					return
				case <-ticker.C:
					if err := task.Action(ctx); err != nil {
						log.Printf("[scheduler] task %q error: %v", task.Name, err)
					}
				}
			}
		}()
	}
}

// Stop signals all goroutines to stop. Safe to call multiple times.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
	})
}
