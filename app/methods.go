package app

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Method struct {
	name         string
	isBlocking   bool
	requiresUser bool
	run          func(a kit.App, r kit.Request, unblock func()) kit.Response
}

func NewMethod(name string, run func(a kit.App, r kit.Request, unblock func()) kit.Response) *Method {
	return &Method{
		name: name,
		run:  run,
	}
}

func (m Method) Name() string {
	return m.name
}

func (m Method) IsBlocking() bool {
	return m.isBlocking
}

func (m Method) RequiresUser() bool {
	return m.requiresUser
}

func (m Method) Run(a kit.App, r kit.Request, unblock func()) kit.Response {
	return m.run(a, r, unblock)
}

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

func (m *methodQueue) Add(method *methodInstance) kit.Error {
	m.lastAction = time.Now()

	if len(m.queue) >= m.maxQueued {
		return kit.AppError{
			Code:    "max_methods_queued",
			Message: "The maximum amount of methods is already running",
		}
	}

	if m.CountAddedSince(60) >= m.maxPerMinute {
		return kit.AppError{
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
		defer func() {
			rawErr := recover()
			if rawErr != nil {
				// Panic occurred, finish with error response.
				resp := &kit.AppResponse{
					Error: kit.AppError{
						Code: "method_panic",
					},
				}
				if err, ok := rawErr.(error); ok {
					resp.Error.AddError(err)
				}

				m.Finish(method, resp)
			}
		}()

		// Run method.
		resp := method.method.Run(m.app, method.request, func() {
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

func (m *SessionManager) QueueMethod(session kit.Session, method *methodInstance) kit.Error {
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
	Objects  []db.Model
	IDs      []string
	Query    db.Query
}

func buildResourceMethodData(app kit.App, rawData interface{}) (*ResourceMethodData, kit.Error) {
	if data, ok := rawData.(ResourceMethodData); ok {
		return &data, nil
	}
	methodData := &ResourceMethodData{}

	data, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, kit.AppError{
			Code:    "invalid_data_map_expected",
			Message: "Data must contain a map/dict",
		}
	}

	// Try to build model objects.
	resourceName, _ := data["resource"].(string)
	if resourceName == "" {
		return nil, kit.AppError{
			Code:    "resource_missing",
			Message: "Data must contain a 'resource' key",
		}
	}

	resource := app.Resource(resourceName)
	if resource == nil {
		return nil, kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v is not registered", resourceName),
		}
	}
	methodData.Resource = resource

	if rawIds, ok := data["ids"].([]interface{}); ok {
		ids := make([]string, 0)
		for index, rawId := range rawIds {
			id, ok := rawId.(string)
			if !ok {
				return nil, kit.AppError{
					Code:    "invalid_id",
					Message: fmt.Sprintf("The %vth id '%v' must be a string", index, rawId),
				}
			}

			ids = append(ids, id)
		}

		methodData.IDs = ids
	}

	if objectData, ok := data["objects"]; ok {
		// Objects key exists, try to parse it.

		if objects, ok := objectData.([]db.Model); ok {
			// Objects are already a model slice.
			methodData.Objects = objects
		} else {
			// Try to unmarshal the data.
			rawObjects, ok := data["objects"].([]interface{})
			if !ok {
				return nil, kit.AppError{
					Code:    "invalid_object_data",
					Message: "Expected array in key 'objects'",
				}
			}

			for index, rawObj := range rawObjects {
				js, err := json.Marshal(rawObj)
				if err != nil {
					return nil, kit.AppError{
						Code:    "json_error",
						Message: err.Error(),
						Errors:  []error{err},
					}
				}

				model := resource.NewModel()
				if err := json.Unmarshal(js, model); err != nil {
					return nil, kit.AppError{
						Code:    "json_unmarshal_error",
						Message: fmt.Sprintf("Could not unmarshal model %v: %v", index, err),
						Errors:  []error{err},
					}
				}

				methodData.Objects = append(methodData.Objects, model)
			}
		}
	}

	// Build query.
	if rawQuery, ok := data["query"].(map[string]interface{}); ok {
		query, err := db.ParseQuery(resourceName, rawQuery)
		if err != nil {
			return nil, kit.AppError{
				Code:    "invalid_query",
				Message: fmt.Sprintf("Error while parsing query: %v", err),
				Errors:  []error{err},
			}
		}
		methodData.Query = query
	}

	return methodData, nil
}

func createMethod() kit.Method {
	return &Method{
		name: "create",
		run: func(a kit.App, r kit.Request, unblock func()) kit.Response {
			methodData, err := buildResourceMethodData(a, r.GetData())
			if err != nil {
				return &kit.AppResponse{
					Error: err,
				}
			}

			if methodData.Objects == nil || len(methodData.Objects) < 1 {
				return kit.NewErrorResponse("no_objects", "No objects sent in data.objects")
			}
			if len(methodData.Objects) != 1 {
				return kit.NewErrorResponse("only_one_object_allowed", "")
			}

			return methodData.Resource.ApiCreate(methodData.Objects[0], r)
		},
	}
}

func updateMethod() kit.Method {
	return &Method{
		name: "update",
		run: func(a kit.App, r kit.Request, unblock func()) kit.Response {
			methodData, err := buildResourceMethodData(a, r.GetData())
			if err != nil {
				return &kit.AppResponse{
					Error: err,
				}
			}

			if methodData.Objects == nil || len(methodData.Objects) < 1 {
				return kit.NewErrorResponse("no_objects", "No objects sent in data.objects")
			}
			if len(methodData.Objects) != 1 {
				return kit.NewErrorResponse("only_one_object_allowed", "")
			}

			return methodData.Resource.ApiUpdate(methodData.Objects[0], r)
		},
	}
}

func deleteMethod() kit.Method {
	return &Method{
		name: "delete",
		run: func(a kit.App, r kit.Request, unblock func()) kit.Response {
			methodData, err := buildResourceMethodData(a, r.GetData())
			if err != nil {
				return &kit.AppResponse{
					Error: err,
				}
			}

			if methodData.IDs == nil || len(methodData.IDs) < 1 {
				return kit.NewErrorResponse("no_ids", "No ids sent in data.ids")
			}
			if len(methodData.IDs) != 1 {
				return kit.NewErrorResponse("only_one_object_allowed", "")
			}

			return methodData.Resource.ApiDelete(methodData.IDs[0], r)
		},
	}
}

func queryMethod() kit.Method {
	return &Method{
		name: "query",
		run: func(a kit.App, r kit.Request, unblock func()) kit.Response {
			methodData, err := buildResourceMethodData(a, r.GetData())
			if err != nil {
				return &kit.AppResponse{
					Error: err,
				}
			}

			if methodData.Query == nil {
				return kit.NewErrorResponse("no_query", "No query sent")
			}

			return methodData.Resource.ApiFind(methodData.Query, r)
		},
	}
}
