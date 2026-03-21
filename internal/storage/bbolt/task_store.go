package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/your-org/gitdex/internal/orchestrator"
)

// TaskStore implements orchestrator.TaskStore using BBolt.
type TaskStore struct {
	db *bbolt.DB
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(db *bbolt.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) SaveTask(task *orchestrator.Task) error {
	if task == nil {
		return fmt.Errorf("cannot save nil task")
	}
	if task.TaskID == "" {
		task.TaskID = orchestrator.GenerateTaskID()
	}
	task.UpdatedAt = time.Now().UTC()

	return s.db.Update(func(tx *bbolt.Tx) error {
		main := tx.Bucket(bucketTasks)
		idx := tx.Bucket(bucketTasksByCorrelationID)
		if main == nil || idx == nil {
			return ErrBucketNotFound
		}

		data, err := jsonMarshal(task)
		if err != nil {
			return err
		}
		if err := main.Put([]byte(task.TaskID), data); err != nil {
			return err
		}

		if task.CorrelationID != "" {
			if err := idx.Put([]byte(task.CorrelationID), []byte(task.TaskID)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *TaskStore) GetTask(taskID string) (*orchestrator.Task, error) {
	var task *orchestrator.Task
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(taskID))
		if v == nil {
			return fmt.Errorf("task %q not found", taskID)
		}
		var t orchestrator.Task
		if err := jsonUnmarshal(v, &t); err != nil {
			return err
		}
		task = &t
		return nil
	})
	return task, err
}

func (s *TaskStore) GetByCorrelationID(corrID string) (*orchestrator.Task, error) {
	var taskID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		idx := tx.Bucket(bucketTasksByCorrelationID)
		if idx == nil {
			return ErrBucketNotFound
		}
		taskID = idx.Get([]byte(corrID))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if taskID == nil {
		return nil, fmt.Errorf("no task found for correlation %q", corrID)
	}
	return s.GetTask(string(taskID))
}

func (s *TaskStore) ListTasks() ([]*orchestrator.Task, error) {
	var result []*orchestrator.Task
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var t orchestrator.Task
			if err := jsonUnmarshal(v, &t); err != nil {
				return err
			}
			result = append(result, &t)
			return nil
		})
	})
	return result, err
}

func (s *TaskStore) UpdateTask(task *orchestrator.Task) error {
	if task == nil {
		return fmt.Errorf("cannot update nil task")
	}
	task.UpdatedAt = time.Now().UTC()

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(task.TaskID)) == nil {
			return fmt.Errorf("task %q not found", task.TaskID)
		}
		data, err := jsonMarshal(task)
		if err != nil {
			return err
		}
		return b.Put([]byte(task.TaskID), data)
	})
}

func (s *TaskStore) AppendEvent(event *orchestrator.TaskEvent) error {
	if event == nil {
		return fmt.Errorf("cannot append nil event")
	}
	if event.EventID == "" {
		event.EventID = orchestrator.GenerateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTaskEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(event)
		if err != nil {
			return err
		}
		return b.Put([]byte(event.EventID), data)
	})
}

func (s *TaskStore) GetEvents(taskID string) ([]*orchestrator.TaskEvent, error) {
	var result []*orchestrator.TaskEvent
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTaskEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var e orchestrator.TaskEvent
			if err := jsonUnmarshal(v, &e); err != nil {
				return err
			}
			if e.TaskID == taskID {
				result = append(result, &e)
			}
			return nil
		})
	})
	return result, err
}
