package tasks

import (
	"time"

	"github.com/theduke/go-apperror"
	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
)

type Service struct {
	Runner
}

var _ kit.TaskService = (*Service)(nil)

func NewService(reg kit.Registry, b db.Backend) *Service {
	var model kit.Model
	if b.HasStringIDs() {
		model = &TaskStrID{}
	} else {
		model = &TaskIntID{}
	}

	s := &Service{}

	runner := NewRunner(reg, b, model)
	s.Runner = *runner

	return s
}

func (s *Service) QueueTask(name string, data interface{}) (string, apperror.Error) {
	rawTask, err := s.backend.CreateModel(s.taskModel.Collection())
	if err != nil {
		return "", err
	}

	task := rawTask.(kit.Task)

	task.SetCreatedAt(time.Now())
	task.SetName(name)
	task.SetData(data)

	if err = s.backend.Create(task); err != nil {
		return "", err
	}
	return task.GetStrID(), nil
}

func (s *Service) GetTask(id string) (kit.Task, apperror.Error) {
	task, err := s.backend.FindOne(s.taskModel.Collection(), id)
	if err != nil || task == nil {
		return nil, err
	}

	return task.(kit.Task), nil
}
