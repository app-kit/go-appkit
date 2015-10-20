package tasks

import (
	"fmt"
	"time"

	"github.com/theduke/go-apperror"
	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
)

type Runner struct {
	registry kit.Registry
	backend  db.Backend

	tasks map[string]kit.TaskSpec

	taskModel kit.Model

	// maximumConcurrentTasks specifies the number of tasks that will be
	// processed concurrently.
	maximumConcurrentTasks int

	// taskCheckInterval specifies the time interval in which new tasks will
	// be fetched from the backend in time.Duration.
	taskCheckInterval time.Duration

	activeTasks map[string]kit.Task

	// progressChan can be used by tasks to report their progress.
	progressChan chan kit.Task

	// finishedChan is used by goroutines that handle a task to signal task completion.
	finishedChan chan kit.Task

	// shutdownChan allows to safely shut down the runner.
	// It will stop starting new tasks, wait until all tasks are finished, and then stop the goroutine.
	// After all tasks are finished, the channel sent to shutdownChan will receive a 'true' value.
	shutdownChan chan chan bool

	// shutdownCompleteChan will receive a bool true after shutdown has completed.
	shutdownCompleteChan chan bool
}

// Ensure that Runner implements appkit.TaskRunner.
var _ kit.TaskRunner = (*Runner)(nil)

func NewRunner(reg kit.Registry, b db.Backend, model kit.Model) *Runner {
	r := &Runner{
		registry:               reg,
		tasks:                  make(map[string]kit.TaskSpec),
		taskModel:              model,
		maximumConcurrentTasks: 20,
		taskCheckInterval:      time.Duration(5) * time.Second,
		activeTasks:            make(map[string]kit.Task),
		progressChan:           make(chan kit.Task),
		finishedChan:           make(chan kit.Task),
		shutdownChan:           make(chan chan bool),
	}
	r.SetBackend(b)

	return r
}

func (r *Runner) SetRegistry(registry kit.Registry) {
	r.registry = registry
}

func (r *Runner) Registry() kit.Registry {
	return r.registry
}

func (r *Runner) SetBackend(backend db.Backend) {
	r.backend = backend

	if backend != nil && r.taskModel != nil && !backend.HasCollection(r.taskModel.Collection()) {
		backend.RegisterModel(r.taskModel)
	}
}

func (r *Runner) Backend() db.Backend {
	return r.backend
}

func (r *Runner) SetMaximumConcurrentTasks(count int) {
	r.maximumConcurrentTasks = count
}

func (r *Runner) MaximumConcurrentTasks() int {
	return r.maximumConcurrentTasks
}

func (r *Runner) SetTaskCheckInterval(duration time.Duration) {
	r.taskCheckInterval = duration
}

func (r *Runner) GetTaskCheckInterval() time.Duration {
	return r.taskCheckInterval
}

func (r *Runner) RegisterTask(spec kit.TaskSpec) {
	if spec.GetName() == "" {
		panic("Trying to register task with empty name")
	}

	r.tasks[spec.GetName()] = spec
}

// GetTaskSpecs returns a slice with all registered tasks.
func (r *Runner) GetTaskSpecs() map[string]kit.TaskSpec {
	return r.tasks
}

func (r *Runner) Run() apperror.Error {
	r.registry.Logger().Debugf("TaskRunner: Launching task runner (max tasks: %v)", r.maximumConcurrentTasks)
	go r.run()
	return nil
}

func (r *Runner) run() {
	lastTaskCheck := time.Time{}

	for {

		// If a shutdown has been ordered,
		// just wait for all tasks to finish.
		if r.shutdownCompleteChan != nil {
			select {
			case task := <-r.finishedChan:
				r.finishTask(task)

				if len(r.activeTasks) < 1 {
					// All tasks have finished.
					// Send shutdown complete signal and return.
					r.registry.Logger().Info("TaskRunner: Shutdown complete")
					r.shutdownCompleteChan <- true
					return
				} else {
					r.registry.Logger().Infof("TaskRunner: Shutting down - waiting for %v remaining tasks to finish.", len(r.activeTasks))
				}
			}
		} else {
			diff := time.Now().Sub(lastTaskCheck)
			if len(r.activeTasks) < r.maximumConcurrentTasks && diff >= r.taskCheckInterval {
				// At least r.taskCheckInterval seconds have passed since the last
				// check, AND less than the maximum concurrent tasks are running,
				// so retrieve new tasks.
				r.startNewTasks()
				lastTaskCheck = time.Now()
			}

			select {
			case task := <-r.progressChan:
				// Update task progress in goroutine to avoid blocking.
				go func(task kit.Task) {
					// Not checking for error since we can't do anything about it anyway.
					r.backend.Update(task)
				}(task)

			case task := <-r.finishedChan:
				r.finishTask(task)

			case c := <-r.shutdownChan:
				r.shutdownCompleteChan = c
				r.registry.Logger().Infof("TaskRunner: Shutting down - waiting for %v remaining tasks to finish", len(r.activeTasks))

			case <-time.After(time.Duration(10) * time.Millisecond):
				// NoOp. Continue loop.
			}
		}

	}
}

func (r *Runner) startNewTasks() {
	tasks, err := r.backend.Q(r.taskModel.Collection()).
		Filter("Complete", false).
		Filter("Running", false).
		Filter("Cancelled", false).
		AndQ(db.Or(
		db.Eq("RunAt", nil),
		db.Lte("RunAt", time.Now()))).
		Limit(r.maximumConcurrentTasks - len(r.activeTasks)).
		Find()

	if err != nil {
		r.registry.Logger().Errorf("TaskRunner: could not fetch new tasks: %v", err)
		return
	}

	for _, rawTask := range tasks {
		task := rawTask.(kit.Task)
		r.runTask(task)
	}
}

func (r *Runner) runTask(task kit.Task) apperror.Error {
	spec := r.tasks[task.GetName()]
	if spec == nil {
		return apperror.New("unknown_task", fmt.Sprintf("The task %v was not registered with the TaskRunner", task.GetName()))
	}

	now := time.Now()
	task.SetStartedAt(&now)
	task.SetIsRunning(true)
	if err := r.backend.Update(task); err != nil {
		return err
	}

	r.activeTasks[task.GetStrID()] = task

	r.registry.Logger().Debugf("TaskRunner: running task %v (task %v, try %v) (%v tasks running)",
		task.GetStrID(),
		task.GetName(),
		task.GetTryCount()+1,
		len(r.activeTasks))

	go func(task kit.Task) {
		result, err, canRetry := spec.GetHandler()(r.registry, task, r.progressChan)

		now := time.Now()
		task.SetFinishedAt(&now)
		task.SetTryCount(task.GetTryCount() + 1)
		task.SetIsRunning(false)

		if err != nil {
			task.SetError(err.Error())

			if !canRetry || task.GetTryCount() >= spec.GetAllowedRetries() {
				task.SetIsComplete(true)
			} else {
				runAt := time.Now().Add(spec.GetRetryInterval())
				task.SetRunAt(&runAt)
			}
		} else {
			task.SetIsComplete(true)
			task.SetIsSuccess(true)
			task.SetResult(result)
		}

		r.finishedChan <- task
	}(task)

	return nil
}

func (r *Runner) finishTask(task kit.Task) {
	if task.IsComplete() {
		if !task.IsSuccess() {
			r.registry.Logger().Debugf("TaskRunner: Task %v failed after %v tries: %v", task.GetStrID(), task.GetTryCount(), task.GetError())
		} else {
			secs := task.GetFinishedAt().Sub(*task.GetStartedAt()).Seconds()
			r.registry.Logger().Debugf("TaskRunner: Task %v completed successfully (%v secs)", task.GetStrID(), secs)
		}
	} else {
		r.Registry().Logger().Debugf("TaskRunner: Task %v(%v) failed, will retry: %v", task.GetStrID(), task.GetName(), task.GetError())
	}

	if err := r.backend.Update(task); err != nil {
		r.registry.Logger().Errorf("TaskRunner: Could not update task: %v", err)
	}

	delete(r.activeTasks, task.GetStrID())

	// Call onComplete handler if specified.
	spec := r.tasks[task.GetName()]
	onComplete := spec.GetOnCompleteHandler()
	if onComplete != nil {
		go onComplete(r.registry, task)
	}
}

func (r *Runner) Shutdown() chan bool {
	c := make(chan bool)
	r.shutdownChan <- c
	return c
}
