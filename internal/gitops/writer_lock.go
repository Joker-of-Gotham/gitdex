package gitops

import (
	"fmt"
	"sync"
	"time"
)

type WriterLock struct {
	mu    sync.Mutex
	locks map[string]lockEntry
}

type lockEntry struct {
	TaskID     string
	AcquiredAt time.Time
}

func NewWriterLock() *WriterLock {
	return &WriterLock{locks: make(map[string]lockEntry)}
}

func lockKey(owner, repo, ref string) string {
	return owner + "/" + repo + "#" + ref
}

func (wl *WriterLock) Acquire(owner, repo, ref, taskID string) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	key := lockKey(owner, repo, ref)
	if existing, ok := wl.locks[key]; ok {
		return fmt.Errorf("writer lock held by task %q since %s", existing.TaskID, existing.AcquiredAt.Format(time.RFC3339))
	}
	wl.locks[key] = lockEntry{TaskID: taskID, AcquiredAt: time.Now().UTC()}
	return nil
}

func (wl *WriterLock) Release(owner, repo, ref, taskID string) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	key := lockKey(owner, repo, ref)
	existing, ok := wl.locks[key]
	if !ok {
		return nil // not locked
	}
	if existing.TaskID != taskID {
		return fmt.Errorf("lock held by task %q, not %q", existing.TaskID, taskID)
	}
	delete(wl.locks, key)
	return nil
}

func (wl *WriterLock) IsLocked(owner, repo, ref string) (bool, string) {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	key := lockKey(owner, repo, ref)
	if entry, ok := wl.locks[key]; ok {
		return true, entry.TaskID
	}
	return false, ""
}
