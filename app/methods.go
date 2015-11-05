package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
	. "github.com/app-kit/go-appkit/app/methods"
)

type methodInstance struct {
	method kit.Method

	request kit.Request

	responder func(kit.Response)

	createdAt  time.Time
	startedAt  time.Time
	finishedAt time.Time

	finishedChannel chan bool

	blocked bool
	stale   bool
}

func NewMethodInstance(m kit.Method, r kit.Request, responder func(kit.Response)) *methodInstance {
	return &methodInstance{
		method:    m,
		request:   r,
		responder: responder,
		blocked:   m.IsBlocking(),
		stale:     false,
	}
}

func (m methodInstance) IsRunning() bool {
	return !m.startedAt.IsZero()
}

type methodQueue struct {
	app *App

	sync.Mutex

	queue map[*methodInstance]bool

	maxQueued    int
	maxRunning   int
	maxPerMinute int
	timeout      int

	lastAction time.Time
}

func newMethodQueue(m *SessionManager) *methodQueue {
	return &methodQueue{
		app:          m.app,
		queue:        make(map[*methodInstance]bool),
		maxQueued:    m.maxQueued,
		maxRunning:   m.maxRunning,
		maxPerMinute: m.maxPerMinute,
		timeout:      m.timeout,
		lastAction:   time.Now(),
	}
}

func (m *methodQueue) TimeSinceActive() int {
	secs := time.Now().Sub(m.lastAction).Seconds()
	return int(secs)
}

func (m *methodQueue) Count() int {
	return len(m.queue)
}

func (m *methodQueue) CountActive() int {
	count := 0
	for method := range m.queue {
		if method.IsRunning() {
			count++
		} else {
			break
		}
	}

	return count
}

func (m *methodQueue) CountAddedSince(seconds int) int {
	now := time.Now()
	count := 0
	for method := range m.queue {
		if now.Sub(method.createdAt).Seconds() <= float64(seconds) {
			count++
		}
	}

	return count
}

func (m *methodQueue) Add(method *methodInstance) apperror.Error {
	m.lastAction = time.Now()

	if len(m.queue) >= m.maxQueued {
		return &apperror.Err{
			Code:    "max_methods_queued",
			Message: "The maximum amount of methods is already running",
		}
	}

	if m.CountAddedSince(60) >= m.maxPerMinute {
		return &apperror.Err{
			Code:    "max_methods_per_minute",
			Message: "You have reached the maximum methods/minute limit.",
		}
	}

	m.Lock()
	m.queue[method] = true
	m.Unlock()

	// Try to process.
	m.Process()

	return nil
}

// Remove methods that have exceeded the timeout.
func (m *methodQueue) PruneStaleMethods() {
	now := time.Now()

	for method := range m.queue {
		if !method.stale && method.IsRunning() && now.Sub(method.startedAt).Seconds() > float64(m.timeout) {
			m.Lock()
			method.stale = true
			m.Unlock()
		}

	}
}

func (m *methodQueue) CanProcess() bool {
	m.PruneStaleMethods()

	running := 0
	for method := range m.queue {
		if method.IsRunning() && !method.stale {
			if method.blocked {
				return false
			}
			running++
		}
	}

	if running >= m.maxRunning {
		return false
	}

	return true
}

func (m *methodQueue) Next() *methodInstance {
	for method := range m.queue {
		if method.IsRunning() || method.stale {
			continue
		}
		return method
	}

	return nil
}

func (m *methodQueue) Process() {
	if !m.CanProcess() {
		return
	}

	m.Lock()
	next := m.Next()

	if next == nil {
		m.Unlock()
		return
	}

	next.startedAt = time.Now()
	m.Unlock()

	go func(method *methodInstance) {

		// Recover from panic.
		/*
			defer func() {
				rawErr := recover()
				if rawErr != nil {
					// Panic occurred, finish with error response.
					resp := &kit.AppResponse{
						Error: kit.AppError{
							Code: "method_panic",
							Data: rawErr,
						},
					}
					if err, ok := rawErr.(error); ok {
						resp.Error.AddError(err)
					}

					m.Finish(method, resp)
				}
			}()
		*/

		// Run method.
		handler := method.method.GetHandler()
		resp := handler(m.app.Registry(), method.request, func() {
			method.blocked = false
		})

		m.Finish(method, resp)
	}(next)
}

func (m *methodQueue) Finish(method *methodInstance, response kit.Response) {
	// Send the response.

	// Recover a panic in the responder.
	defer func() {
		err := recover()
		if err != nil {
			// Responder paniced!

			// Remove method from queue.
			m.Lock()
			delete(m.queue, method)
			m.Unlock()

			method.finishedAt = time.Now()
		}
	}()

	// Send the response.
	method.responder(response)
	if method.finishedChannel != nil {
		method.finishedChannel <- true
	}

	// Remove method from queue.
	m.Lock()
	delete(m.queue, method)
	m.Unlock()

	method.finishedAt = time.Now()

	// Try to run queued methods.
	m.Process()
}

type SessionManager struct {
	app *App

	sync.Mutex

	queues map[kit.Session]*methodQueue

	maxQueued    int
	maxRunning   int
	maxPerMinute int
	timeout      int

	sessionTimeout int
	pruneInterval  int
}

func NewSessionManager(app *App) *SessionManager {
	return &SessionManager{
		app:    app,
		queues: make(map[kit.Session]*methodQueue),

		maxQueued:    app.Config().UInt("methods.maxQueued", 30),
		maxRunning:   app.Config().UInt("methods.maxRunning", 5),
		maxPerMinute: app.Config().UInt("methods.maxPerMinute", 100),
		timeout:      app.Config().UInt("methods.timeout", 30),

		sessionTimeout: app.Config().UInt("sessions.sessionTimeout", 60*4),
		pruneInterval:  app.Config().UInt("sessions.pruneInterval", 60*5),
	}
}

func (m *SessionManager) QueueMethod(session kit.Session, method *methodInstance) apperror.Error {
	queue := m.queues[session]
	if queue == nil {
		m.Lock()
		m.queues[session] = newMethodQueue(m)
		m.Unlock()
		queue = m.queues[session]
	}

	err := queue.Add(method)
	if err != nil {
		return err
	}

	return nil
}

func (m *SessionManager) Prune() {
	m.Lock()
	for session, queue := range m.queues {
		if queue.Count() == 0 && queue.TimeSinceActive() >= m.sessionTimeout {
			delete(m.queues, session)
		}
	}
	m.Unlock()
}

func (m *SessionManager) Run() {
	go func() {
		m.Prune()
		time.Sleep(time.Duration(m.pruneInterval) * time.Second)
	}()
}

type ResourceMethodData struct {
	Resource kit.Resource
	Objects  []kit.Model
	Ids      []string
	Query    db.Query
}

var createMethod kit.Method = &Method{
	Name:     "create",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		models := r.GetTransferData().GetModels()
		if len(models) == 0 {
			return kit.NewErrorResponse("no_model", "No model was found in the request.")
		} else if len(models) > 1 {
			return kit.NewErrorResponse("multiple_models", "Request contained more than one model.")
		}

		res := registry.Resource(models[0].Collection())
		if res == nil || !res.IsPublic() {
			return kit.NewErrorResponse("unknown_collection", fmt.Sprintf("The collection %v does not exist", models[0].Collection()))
		}

		return res.ApiCreate(models[0], r)
	},
}

var updateMethod kit.Method = &Method{
	Name:     "update",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		models := r.GetTransferData().GetModels()
		if len(models) == 0 {
			return kit.NewErrorResponse("no_model", "No model was found in the request.")
		} else if len(models) > 1 {
			return kit.NewErrorResponse("multiple_models", "Request contained more than one model.")
		}

		res := registry.Resource(models[0].Collection())
		if res == nil || !res.IsPublic() {
			return kit.NewErrorResponse("unknown_collection", fmt.Sprintf("The collection %v does not exist", models[0].Collection()))
		}

		return res.ApiUpdate(models[0], r)
	},
}

var deleteMethod kit.Method = &Method{
	Name:     "delete",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		data, _ := r.GetData().(map[string]interface{})
		if data == nil {
			return kit.NewErrorResponse("no_data", "No request data.")
		}
		collection, _ := data["collection"].(string)
		id, _ := data["id"].(string)

		if collection == "" {
			return kit.NewErrorResponse("no_collection", "Expected 'collection' key in data")
		} else if id == "" {
			return kit.NewErrorResponse("no_id", "Expected 'id' key with string value in data")
		}

		res := registry.Resource(collection)
		if res == nil || !res.IsPublic() {
			return kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The collection %v does not exist", collection))
		}

		return res.ApiDelete(id, r)
	},
}

var queryMethod kit.Method = &Method{
	Name:     "query",
	Blocking: false,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		// Build query.
		data, _ := r.GetData().(map[string]interface{})
		if data == nil {
			return kit.NewErrorResponse("no_data", "No request data.", true)
		}

		rawQuery, _ := data["query"].(map[string]interface{})
		if rawQuery == nil {
			return kit.NewErrorResponse("no_query", "No query in request data.", true)
		}

		collection, _ := rawQuery["collection"].(string)
		if collection == "" {
			return kit.NewErrorResponse("empty_collection", "No collection in query", true)
		}

		resource := registry.Resource(collection)
		if resource == nil {
			return kit.NewErrorResponse("unknown_collection", "Unknown collection", true)
		}

		query, err := db.ParseQuery(resource.Backend(), rawQuery)
		if err != nil {
			if err.IsPublic() {
				return kit.NewErrorResponse(err)
			} else {
				return kit.NewErrorResponse("invalid_query", err)
			}
		}

		res := registry.Resource(query.GetCollection())
		if res == nil {
			return kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The collection %v does not exist", query.GetCollection()))
		}

		return res.ApiFind(query, r)
	},
}

var findOneMethod kit.Method = &Method{
	Name:     "find_one",
	Blocking: false,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		data, _ := r.GetData().(map[string]interface{})
		if data == nil {
			return kit.NewErrorResponse("no_data", "No request data.")
		}
		collection, _ := data["collection"].(string)
		id, _ := data["id"].(string)

		if collection == "" {
			return kit.NewErrorResponse("no_collection", "Expected 'collection' key in data")
		} else if id == "" {
			return kit.NewErrorResponse("no_id", "Expected 'id' key with string value in data")
		}

		res := registry.Resource(collection)
		if res == nil || !res.IsPublic() {
			return kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The collection %v does not exist", collection))
		}

		return res.ApiFindOne(id, r)
	},
}
