package tasks_test

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/theduke/go-apperror"
	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app"
	"github.com/theduke/go-dukedb/backends/memory"

	. "github.com/theduke/go-appkit/tasks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FailingTaskSpec struct {
	TaskSpec

	FailCount int
}

func buildRegistry() kit.Registry {
	registry := app.NewRegistry()

	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	registry.SetLogger(logger)

	backend := memory.New()
	registry.AddBackend(backend)

	service := NewService(registry, backend)
	registry.SetTaskService(service)

	runner := service.Runner

	t1 := &TaskSpec{
		Name:           "task1",
		RetryInterval:  time.Duration(2) * time.Second,
		AllowedRetries: 3,
		Handler: func(reg kit.Registry, data interface{}) (interface{}, apperror.Error, bool) {
			v, ok := data.(int64)
			if !ok {
				return nil, apperror.New("invalid_data"), false
			}

			return v * 2, nil, false
		},
	}
	runner.RegisterTask(t1)

	failCounter := 2
	t2 := &TaskSpec{
		Name:           "task2",
		RetryInterval:  time.Duration(1) * time.Second,
		AllowedRetries: 3,
		Handler: func(reg kit.Registry, data interface{}) (interface{}, apperror.Error, bool) {
			reg.Logger().Infof("runCounter: %v", failCounter)
			if failCounter > 0 {
				failCounter -= 1
				return nil, apperror.New("failed"), true
			} else {
				return data.(int64) * 2, nil, false
			}
		},
	}
	runner.RegisterTask(t2)

	t3 := &TaskSpec{
		Name:           "task_sleeping",
		RetryInterval:  time.Duration(2) * time.Second,
		AllowedRetries: 3,
		Handler: func(reg kit.Registry, data interface{}) (interface{}, apperror.Error, bool) {
			v := data.(int)
			time.Sleep(time.Duration(v) * time.Millisecond)

			return 1, nil, false
		},
	}
	runner.RegisterTask(t3)

	return registry
}

var _ = Describe("Service", func() {
	registry := buildRegistry()
	service := registry.TaskService()
	runner := service.(*Service).Runner

	runner.SetTaskCheckInterval(time.Duration(10) * time.Millisecond)
	runner.SetMaximumConcurrenTasks(2)

	runner.Run()

	It("Should start and finish a task", func() {
		id, err := service.QueueTask("task1", int64(22))
		Expect(err).ToNot(HaveOccurred())
		Expect(id).ToNot(BeEmpty())

		for i := 0; i < 10; i++ {
			time.Sleep(time.Duration(100) * time.Millisecond)
			t, err := service.GetTask(id)
			Expect(err).ToNot(HaveOccurred())

			if t.IsComplete() {
				Expect(t.IsSuccess()).To(BeTrue())
				result := t.GetResult().(int64)
				Expect(result).To(Equal(int64(44)))
				return
			}
		}

		Fail("Task did not complete")
	})

	It("Should fail immediately with canRetry=false", func() {
		id, err := service.QueueTask("task1", 22)
		Expect(err).ToNot(HaveOccurred())
		Expect(id).ToNot(BeEmpty())

		for i := 0; i < 10; i++ {
			time.Sleep(time.Duration(100) * time.Millisecond)
			t, err := service.GetTask(id)
			Expect(err).ToNot(HaveOccurred())

			if t.IsComplete() {
				Expect(t.IsSuccess()).To(BeFalse())
				return
			}
		}

		Fail("Task did not complete")
	})

	It("Should retry tasks", func() {
		id, err := service.QueueTask("task2", int64(22))
		Expect(err).ToNot(HaveOccurred())
		Expect(id).ToNot(BeEmpty())

		for i := 0; i < 10; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			t, err := service.GetTask(id)
			Expect(err).ToNot(HaveOccurred())

			if t.GetTryCount() == 3 {
				Expect(t.IsSuccess()).To(BeTrue())
				Expect(t.GetResult().(int64)).To(Equal(int64(44)))
				return
			}
		}

		Fail("Task did not complete")
	})

	It("Should process queue of tasks", func() {
		ids := make([]string, 0)
		for i := 0; i < 10; i++ {
			id, err := service.QueueTask("task_sleeping", 250)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).ToNot(BeEmpty())

			ids = append(ids, id)
		}

		time.Sleep(time.Duration(5) * time.Second)

		for i := 0; i < 10; i++ {
			t, err := service.GetTask(ids[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(t.IsSuccess()).To(BeTrue())
		}
	})

})
