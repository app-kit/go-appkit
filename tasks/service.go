package tasks

import (
	"time"

	kit "github.com/app-kit/go-appkit"
	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
)

type Service struct {
	Runner
}

var _ kit.TaskService = (*Service)(nil)

func NewService(reg kit.Registry, b db.Backend) *Service {
	var model kit.Model
	if b.HasStringIds() {
		model = &TaskStrId{}
	} else {
		model = &TaskIntId{}
	}

	s := &Service{}

	runner := NewRunner(reg, b, model)
	s.Runner = *runner

	return s
}

func (s *Service) Queue(task kit.Task) apperror.Error {
	task.SetCreatedAt(time.Now())

	if task.GetName() == "" {
		return apperror.New("task_name_empty", "Can't queue a task without a name")
	}

	if err := s.backend.Create(task); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetTask(id string) (kit.Task, apperror.Error) {
	task, err := s.backend.FindOne(s.taskModel.Collection(), id)
	if err != nil || task == nil {
		return nil, err
	}

	return task.(kit.Task), nil
}
