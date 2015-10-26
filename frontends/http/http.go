package http

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/frontends"
)

type Frontend struct {
	registry kit.Registry
	debug    bool

	serverErrorHandler kit.AfterRequestMiddleware
	notFoundHandler    kit.RequestHandler
	router             *httprouter.Router

	beforeMiddlewares []kit.RequestHandler
	afterMiddlewares  []kit.AfterRequestMiddleware
}

// Ensure that Frontend implements kit.HttpFrontend.
var _ kit.HttpFrontend = (*Frontend)(nil)

func New(registry kit.Registry) *Frontend {
	f := &Frontend{
		registry: registry,

		notFoundHandler: notFoundHandler,

		beforeMiddlewares: make([]kit.RequestHandler, 0),
		afterMiddlewares:  make([]kit.AfterRequestMiddleware, 0),

		router: httprouter.New(),
	}

	f.RegisterBeforeMiddleware(frontends.RequestTraceMiddleware)
	f.RegisterBeforeMiddleware(UnserializeRequestMiddleware)
	f.RegisterBeforeMiddleware(AuthenticationMiddleware)

	f.RegisterAfterMiddleware(ServerErrorMiddleware)
	f.RegisterAfterMiddleware(frontends.SerializeResponseMiddleware)
	f.RegisterAfterMiddleware(MarshalResponseMiddleware)
	f.RegisterAfterMiddleware(frontends.RequestTraceAfterMiddleware)
	f.RegisterAfterMiddleware(frontends.RequestLoggerMiddleware)

	return f
}

func (Frontend) Name() string {
	return "http"
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
 * Router.
 */

func (f *Frontend) Router() *httprouter.Router {
	return f.router
}

/**
 * Serve files.
 */

func (f *Frontend) ServeFiles(route string, path string) {
	f.Logger().Debugf("Serving files from directory '%v' at route '%v'", path, route)

	server := http.FileServer(http.Dir(path))
	f.router.GET(route+"/*path", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Fix the url.
		r.URL.Path = params.ByName("path")
		server.ServeHTTP(w, r)
	})
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

/**
 * Http handlers.
 */

func (f *Frontend) NotFoundHandler() kit.RequestHandler {
	return f.notFoundHandler
}

func (f *Frontend) SetNotFoundHandler(x kit.RequestHandler) {
	f.notFoundHandler = x
}

func (f *Frontend) RegisterHttpHandler(method, path string, handler kit.RequestHandler) {
	f.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		HttpHandler(w, r, params, f.registry, handler)
	})
}

/**
 * Run methods.
 */

func (f *Frontend) Init() apperror.Error {

	// Install not found handler.
	f.router.NotFound = &HttpHandlerStruct{
		Registry: f.registry,
		Handler:  f.notFoundHandler,
	}

	// Install handler for index.
	indexTpl, err := getIndexTpl(f.registry)
	if err != nil {
		f.Logger().Panic(err)
	}

	f.router.GET("/", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		HttpHandler(w, r, params, f.registry, func(kit.Registry, kit.Request) (kit.Response, bool) {
			return &kit.AppResponse{
				RawData: indexTpl,
			}, false
		})
	})

	// Serve files routes.

	serveFiles := f.registry.Config().UMap("serveFiles")
	for route := range serveFiles {
		path, ok := serveFiles[route].(string)
		if !ok {
			f.Logger().Error("Config error: serveFiles configuration invalid: Must be map/dictionary with paths")
			continue
		}
		f.ServeFiles(route, path)
	}

	return nil
}

func (f *Frontend) Start() apperror.Error {
	url := f.registry.Config().UString("host", "localhost") + ":" + f.registry.Config().UString("port", "8000")
	f.Logger().Debugf("Serving on %v", url)

	go func() {
		err2 := http.ListenAndServe(url, f.router)
		if err2 != nil {
			f.Logger().Panicf("Could not start server: %v\n", err2)
		}
	}()

	return nil
}

func (f *Frontend) Shutdown() (shutdownChan chan bool, err apperror.Error) {
	return nil, nil
}
