package tasks

import (
	"time"

	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
)

type Task struct {
	Name     string      `db:"not-null"`
	Data     interface{} `db:"marshal"`
	RunAt    *time.Time
	Priority int
	Progress int

	CreatedAt time.Time

	Cancelled bool

	Result interface{} `db:"marshal"`

	TryCount int

	StartedAt  *time.Time
	FinishedAt *time.Time

	Running  bool
	Complete bool
	Success  bool

	Error string
	Log   string
}

func (Task) Collection() string {
	return "tasks"
}

// GetName Returns the name of the task (see @TaskSpec).
func (t Task) GetName() string {
	return t.Name
}

func (t *Task) SetName(name string) {
	t.Name = name
}

func (t *Task) GetPriority() int {
	return t.Priority
}

func (t *Task) SetPriority(x int) {
	t.Priority = x
}

func (t *Task) GetProgress() int {
	return t.Progress
}

func (t *Task) SetProgress(x int) {
	t.Progress = x
}

// GetData returns the data associated with the task.
func (t Task) GetData() interface{} {
	return t.Data
}

func (t *Task) SetData(data interface{}) {
	t.Data = data
}

// GetResult returns the result data omitted by the task.
func (t Task) GetResult() interface{} {
	return t.Result
}

// SetResult sets the result data omitted by the task.
func (t *Task) SetResult(result interface{}) {
	t.Result = result
}

func (t *Task) GetCreatedAt() time.Time {
	return t.CreatedAt
}

func (t *Task) IsCancelled() bool {
	return t.Cancelled
}

func (t *Task) SetIsCancelled(x bool) {
	t.Cancelled = x
}

func (t *Task) SetCreatedAt(tm time.Time) {
	t.CreatedAt = tm
}

func (t *Task) GetRunAt() *time.Time {
	return t.RunAt
}

func (t *Task) SetRunAt(x *time.Time) {
	t.RunAt = x
}

// TryCount returns the number of times the task has been tried.
func (t Task) GetTryCount() int {
	return t.TryCount
}

func (t *Task) SetTryCount(count int) {
	t.TryCount = count
}

// StartedAt returns a time if the task was started, or zero value otherwise.
func (t Task) GetStartedAt() *time.Time {
	return t.StartedAt
}

func (t *Task) SetStartedAt(tm *time.Time) {
	t.StartedAt = tm
}

// FinishedAt returns the time the task was finished, or zero value.
func (t Task) GetFinishedAt() *time.Time {
	return t.FinishedAt
}

func (t *Task) IsRunning() bool {
	return t.Running
}

func (t *Task) SetIsRunning(x bool) {
	t.Running = x
}

func (t *Task) SetFinishedAt(tm *time.Time) {
	t.FinishedAt = tm
}

func (t Task) IsSuccess() bool {
	return t.Success
}

func (t *Task) SetIsSuccess(flag bool) {
	t.Success = flag
}

func (t Task) IsComplete() bool {
	return t.Complete
}

func (t *Task) SetIsComplete(flag bool) {
	t.Complete = flag
}

// GetError returns the error that occured on the last try, or nil if none.
func (t Task) GetError() string {
	return t.Error
}

func (t *Task) SetError(err string) {
	t.Error = err
}

// Returns the log messages the last task run produced.
func (t Task) GetLog() string {
	return t.Log
}

func (t *Task) SetLog(log string) {
	t.Log = log
}

type TaskIntID struct {
	db.IntIDModel
	Task

	UserID uint64
}

// Ensure that TaskIntID implements appkit.Task.
var _ kit.Task = (*TaskIntID)(nil)

func (t TaskIntID) GetUserID() interface{} {
	return t.UserID
}

func (t TaskIntID) SetUserID(id interface{}) {
	t.UserID = id.(uint64)
}

type TaskStrID struct {
	db.StrIDModel
	Task

	UserID string
}

// Ensure that TaskStrID implements appkit.Task.
var _ kit.Task = (*TaskStrID)(nil)

func (t TaskStrID) GetUserID() interface{} {
	return t.UserID
}

func (t TaskStrID) SetUserID(id interface{}) {
	t.UserID = id.(string)
}
