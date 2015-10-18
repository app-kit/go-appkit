package tasks

import (
	"time"

	kit "github.com/theduke/go-appkit"
)

type TaskSpec struct {
	Name           string
	AllowedRetries int
	RetryInterval  time.Duration
	Handler        kit.TaskHandler
}

// GetName returns a unique name for the task.
func (s TaskSpec) GetName() string {
	return s.Name
}

// GetAllowedRetries returns the number of allowed retries.
func (s TaskSpec) GetAllowedRetries() int {
	return s.AllowedRetries
}

// GetRetryInterval returns the time in seconds that must pass before a
// retry is attempted.
func (s TaskSpec) GetRetryInterval() time.Duration {
	return s.RetryInterval
}

// GetHandler returns the TaskHandler function that will execute the task.
func (s TaskSpec) GetHandler() kit.TaskHandler {
	return s.Handler
}
