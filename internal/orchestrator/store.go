package orchestrator

import (
	"fmt"
	"sync"
	"time"
)

type TaskStore interface {
	SaveTask(task *Task) error
	GetTask(taskID string) (*Task, error)
	GetByCorrelationID(corrID string) (*Task, error)
	ListTasks() ([]*Task, error)
	UpdateTask(task *Task) error
	AppendEvent(event *TaskEvent) error
	GetEvents(taskID string) ([]*TaskEvent, error)
}

type MemoryTaskStore struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	events map[string][]*TaskEvent
}

func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{
		tasks:  make(map[string]*Task),
		events: make(map[string][]*TaskEvent),
	}
}

func (s *MemoryTaskStore) SaveTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("cannot save nil task")
	}
	if task.TaskID == "" {
		task.TaskID = GenerateTaskID()
	}
	task.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *task
	cp.Steps = make([]StepResult, len(task.Steps))
	copy(cp.Steps, task.Steps)
	s.tasks[task.TaskID] = &cp
	return nil
}

func (s *MemoryTaskStore) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}
	cp := *t
	cp.Steps = make([]StepResult, len(t.Steps))
	copy(cp.Steps, t.Steps)
	return &cp, nil
}

func (s *MemoryTaskStore) GetByCorrelationID(corrID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tasks {
		if t.CorrelationID == corrID {
			cp := *t
			cp.Steps = make([]StepResult, len(t.Steps))
			copy(cp.Steps, t.Steps)
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("no task found for correlation %q", corrID)
}

func (s *MemoryTaskStore) ListTasks() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		cp := *t
		cp.Steps = make([]StepResult, len(t.Steps))
		copy(cp.Steps, t.Steps)
		result = append(result, &cp)
	}
	return result, nil
}

func (s *MemoryTaskStore) UpdateTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("cannot update nil task")
	}
	task.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[task.TaskID]; !ok {
		return fmt.Errorf("task %q not found", task.TaskID)
	}
	cp := *task
	cp.Steps = make([]StepResult, len(task.Steps))
	copy(cp.Steps, task.Steps)
	s.tasks[task.TaskID] = &cp
	return nil
}

func (s *MemoryTaskStore) AppendEvent(event *TaskEvent) error {
	if event == nil {
		return fmt.Errorf("cannot append nil event")
	}
	if event.EventID == "" {
		event.EventID = GenerateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *event
	s.events[event.TaskID] = append(s.events[event.TaskID], &cp)
	return nil
}

func (s *MemoryTaskStore) GetEvents(taskID string) ([]*TaskEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[taskID]
	result := make([]*TaskEvent, len(events))
	for i, e := range events {
		cp := *e
		result[i] = &cp
	}
	return result, nil
}
