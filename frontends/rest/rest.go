package rest

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"

	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
	apphttp "github.com/app-kit/go-appkit/frontends/http"
)

type Frontend struct {
	registry kit.Registry
	debug    bool
}

// Ensure that Frontend implements appkit.Frontend.
var _ kit.Frontend = (*Frontend)(nil)

func New(registry kit.Registry) *Frontend {
	f := &Frontend{
		registry: registry,
	}

	return f
}

func (Frontend) Name() string {
	return "rest"
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
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) SetBeforeMiddlewares(middlewares []kit.RequestHandler) {
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) BeforeMiddlewares() []kit.RequestHandler {
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) RegisterAfterMiddleware(middleware kit.AfterRequestMiddleware) {
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) SetAfterMiddlewares(middlewares []kit.AfterRequestMiddleware) {
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) AfterMiddlewares() []kit.AfterRequestMiddleware {
	panic("REST frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) Init() apperror.Error {
	// Register route for method calls.
	methodHandler := func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		var response kit.Response

		responder := func(r kit.Response) {
			response = r
		}

		method := r.GetContext().String("name")

		finishedChannel, err := registry.App().RunMethod(method, r, responder, true)
		if err != nil {
			return kit.NewErrorResponse(err), false
		}
		<-finishedChannel

		return response, false
	}

	httpFrontend := f.registry.HttpFrontend()
	if httpFrontend == nil {
		return apperror.New("http_frontend_required", "The JSONAPI frontend relies on the HTTP frontend, which was not found")
	}

	// Handle options requests.
	httpFrontend.Router().OPTIONS("/api/method/:name", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		apphttp.HttpHandler(w, r, params, f.registry, func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
			return &kit.AppResponse{}, false
		})
	})
	// Handle the method request.
	httpFrontend.Router().POST("/api/method/:name", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		apphttp.HttpHandler(w, r, params, f.registry, methodHandler)
	})

	return nil
}

func (f *Frontend) Start() apperror.Error {
	return nil
}

func (f *Frontend) Shutdown() (shutdownChan chan bool, err apperror.Error) {
	return nil, nil
}
