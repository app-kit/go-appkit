package wamp

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/theduke/go-apperror"
	"gopkg.in/jcelliott/turnpike.v2"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/frontends"
)

func UnserializerMiddleware(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	serializer := registry.DefaultSerializer()

	// Try to find custom serializer.
	data, ok := request.GetData().(map[string]interface{})
	if ok {
		name, ok := data["request_serializer"].(string)
		if ok {
			s := registry.Serializer(name)
			if s == nil {
				resp := kit.NewErrorResponse("unknown_request_serializer", fmt.Sprintf("The given request serializer %v does not exist", name))
				return resp, false
			} else {
				serializer = s
			}
		}
	}

	if err := serializer.UnserializeRequest(request.GetData(), request); err != nil {
		return kit.NewErrorResponse(err), false
	}

	return nil, false
}

type Frontend struct {
	registry kit.Registry
	debug    bool

	beforeMiddlewares []kit.RequestHandler
	afterMiddlewares  []kit.AfterRequestMiddleware

	server *turnpike.WebsocketServer
	client *turnpike.Client

	sessions map[uint]kit.Session
}

// Ensure that Frontend implements appkit.Frontend.
var _ kit.Frontend = (*Frontend)(nil)

func New(registry kit.Registry) *Frontend {
	conf := registry.Config()

	f := &Frontend{
		registry:          registry,
		debug:             conf.UBool("frontends.wamp.debug", false),
		beforeMiddlewares: make([]kit.RequestHandler, 0),
		afterMiddlewares:  make([]kit.AfterRequestMiddleware, 0),
		sessions:          make(map[uint]kit.Session),
	}

	f.RegisterBeforeMiddleware(frontends.RequestTraceMiddleware)
	f.RegisterBeforeMiddleware(UnserializerMiddleware)

	f.RegisterAfterMiddleware(frontends.SerializeResponseMiddleware)
	f.RegisterAfterMiddleware(frontends.RequestTraceAfterMiddleware)
	f.RegisterAfterMiddleware(frontends.RequestLoggerMiddleware)

	return f
}

func (Frontend) Name() string {
	return "wamp"
}

func (f *Frontend) Registry() kit.Registry {
	return f.registry
}

func (f *Frontend) SetRegistry(x kit.Registry) {
	f.registry = x
}

func (f *Frontend) Debug() bool {
	return f.debug
}

func (f *Frontend) SetDebug(x bool) {
	f.debug = x
}

func (f *Frontend) Logger() *logrus.Logger {
	return f.registry.Logger()
}

/**
 * Middlewares.
 */

func (f *Frontend) RegisterBeforeMiddleware(handler kit.RequestHandler) {
	f.beforeMiddlewares = append(f.beforeMiddlewares, handler)
}

func (f *Frontend) SetBeforeMiddlewares(middlewares []kit.RequestHandler) {
	f.beforeMiddlewares = middlewares
}

func (f *Frontend) BeforeMiddlewares() []kit.RequestHandler {
	return f.beforeMiddlewares
}

func (f *Frontend) RegisterAfterMiddleware(middleware kit.AfterRequestMiddleware) {
	f.afterMiddlewares = append(f.afterMiddlewares, middleware)
}

func (f *Frontend) SetAfterMiddlewares(middlewares []kit.AfterRequestMiddleware) {
	f.afterMiddlewares = middlewares
}

func (f *Frontend) AfterMiddlewares() []kit.AfterRequestMiddleware {
	return f.afterMiddlewares
}

func (f *Frontend) Init() apperror.Error {
	httpFrontend := f.registry.HttpFrontend()
	if httpFrontend == nil {
		return apperror.New("http_frontend_required", "The JSONAPI frontend relies on the HTTP frontend, which was not found")
	}

	conf := f.Registry().Config()

	realm := conf.UString("frontends.wamp.realm", "appkit")
	path := conf.UString("frontends.wamp.path", "/api/wamp")

	// Build turnpike server and client.

	if f.debug {
		turnpike.Debug()
	}
	server := turnpike.NewBasicWebsocketServer(realm)

	// Install session open/close callbacks.

	userService := f.registry.UserService()

	server.AddSessionOpenCallback(func(id uint, realm string) {
		session, err := userService.StartSession(nil, "wamp")
		if err != nil {
			// Todo: figure out how to handle an error.
		}

		f.sessions[id] = session
	})

	server.AddSessionCloseCallback(func(id uint, realm string) {
		delete(f.sessions, id)
	})

	// Register websocket handler.
	httpFrontend.Router().Handle("GET", path, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		req.Header["Origin"] = nil
		server.ServeHTTP(w, req)
	})

	f.server = server

	// Build local client.
	client, err := server.GetLocalClient(realm, nil)
	if err != nil {
		return apperror.Wrap(err, "turnpike_local_client_error")
	}
	f.client = client

	return nil
}

func convertResponse(response kit.Response) *turnpike.CallResult {
	result := &turnpike.CallResult{}

	if response.GetError() != nil {
		result.Err = turnpike.URI(response.GetError().GetCode())
	}

	result.Kwargs = map[string]interface{}{
		"data":   response.GetData(),
		"meta":   response.GetMeta(),
		"errors": []error{response.GetError()},
	}

	return result
}

func (f *Frontend) registerMethod(method kit.Method) {
	f.client.Register(method.GetName(), func(args []interface{}, kwargs map[string]interface{}, details map[string]interface{}) (result *turnpike.CallResult) {
		methodName := method.GetName()
		fmt.Printf("WAMP method %v |\n data: %v |\n details: %v\n", methodName, kwargs, details)

		request := kit.NewRequest()
		request.SetFrontend("wamp")
		request.SetPath("/method/" + methodName)
		request.SetData(kwargs)

		var response kit.Response

		// Find session.
		sessionId := uint(details["session_id"].(turnpike.ID))
		session := f.sessions[sessionId]
		if session == nil {
			s, err := f.registry.UserService().StartSession(nil, "wamp")
			if err != nil {
				response = kit.NewErrorResponse(err)
			} else {
				f.sessions[sessionId] = s
				session = s
			}
		}

		request.SetSession(session)
		if session.GetUser() != nil {
			request.SetUser(session.GetUser())
		}

		// Run before middlewares.
		for _, middleware := range f.beforeMiddlewares {
			resp, skip := middleware(f.registry, request)
			if skip {
				panic("WAMP frontend middlewares do not support skipping.")
			} else if resp != nil {
				response = resp
				break
			}
		}

		// Run the method.
		if response == nil {
			responder := func(r kit.Response) {
				response = r
			}

			finishedChannel, err := f.registry.App().RunMethod(methodName, request, responder, true)
			if err != nil {
				response = kit.NewErrorResponse(err)
			} else {
				<-finishedChannel
			}
		}

		// Run after middlewares.
		for _, middleware := range f.afterMiddlewares {
			resp, skip := middleware(f.registry, request, response)
			if skip {
				panic("WAMP frontend middlewares do not support skipping.")
			}
			if resp != nil {
				response = resp
			}
		}

		return &turnpike.CallResult{Kwargs: response.GetData().(map[string]interface{})}
	}, nil)
}

func (f *Frontend) Start() apperror.Error {
	// Register methods.
	for _, method := range f.registry.Methods() {
		f.registerMethod(method)
	}

	return nil
}

func (f *Frontend) Shutdown() (shutdownChan chan bool, err apperror.Error) {
	return nil, nil
}
