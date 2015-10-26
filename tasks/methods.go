package tasks

import (
	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/app/methods"
)

var RetryTaskMethod = &methods.Method{
	Name:     "task.retry",
	Blocking: false,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		user := r.GetUser()
		if user == nil {
			return kit.NewErrorResponse("not_authenticated", true)
		}

		taskId := r.GetData()
		if taskId == nil {
			return kit.NewErrorResponse("invalid_task_id", "Expected 'data' to be the task ID.", true)
		}

		backend := registry.DefaultBackend()
		rawTask, err := backend.FindOne("tasks", taskId)
		if err != nil {
			return kit.NewErrorResponse(err)
		} else if rawTask == nil {
			return kit.NewErrorResponse("not_found", "Task does not exist.")
		}

		task := rawTask.(kit.Task)

		// Permissions check.
		if !(user.HasRole("admin") || user.GetID() == task.GetUserID()) {
			return kit.NewErrorResponse("permission_denied")
		}

		if task.IsSuccess() {
			return kit.NewErrorResponse("task_succeeded", "Can't retry a succeeded task")
		} else if !task.IsComplete() {
			return kit.NewErrorResponse("task_not_complete", "Can't retry a task that has not completed yet.")
		}

		task.SetIsComplete(false)
		task.SetTryCount(0)
		task.SetRunAt(nil)

		if err := backend.Update(task); err != nil {
			return kit.NewErrorResponse(err)
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"success": true},
		}
	},
}

var CancelTaskMethod = &methods.Method{
	Name:     "task.cancel",
	Blocking: false,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		user := r.GetUser()
		if user == nil {
			return kit.NewErrorResponse("not_authenticated", true)
		}

		taskId := r.GetData()
		if taskId == nil {
			return kit.NewErrorResponse("invalid_task_id", "Expected 'data' to be the task ID.", true)
		}

		backend := registry.DefaultBackend()
		rawTask, err := backend.FindOne("tasks", taskId)
		if err != nil {
			return kit.NewErrorResponse(err)
		} else if rawTask == nil {
			return kit.NewErrorResponse("not_found", "Task does not exist.")
		}

		task := rawTask.(kit.Task)

		// Permissions check.
		if !(user.HasRole("admin") || user.GetID() == task.GetUserID()) {
			return kit.NewErrorResponse("permission_denied")
		}

		if task.IsComplete() {
			return kit.NewErrorResponse("task_complete", "Can't cancel a completed task.")
		} else if task.IsRunning() {
			return kit.NewErrorResponse("task_running", "Can't cancel a running task.")
		}

		if err := backend.Update(task); err != nil {
			return kit.NewErrorResponse(err)
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"success": true},
		}
	},
}
