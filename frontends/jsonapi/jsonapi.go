package jsonapi

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/theduke/go-apperror"

	kit "github.com/theduke/go-appkit"
)

type Frontend struct {
	registry kit.Registry
	debug    bool
}

// Ensure Frontend implements kit.Frontend.
var _ kit.Frontend = (*Frontend)(nil)

func New(registry kit.Registry) *Frontend {
	return &Frontend{
		registry: registry,
	}
}

func (f *Frontend) Name() string {
	return "jsonapi"
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
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) SetBeforeMiddlewares(middlewares []kit.RequestHandler) {
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) BeforeMiddlewares() []kit.RequestHandler {
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) RegisterAfterMiddleware(middleware kit.AfterRequestMiddleware) {
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) SetAfterMiddlewares(middlewares []kit.AfterRequestMiddleware) {
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) AfterMiddlewares() []kit.AfterRequestMiddleware {
	panic("JSONAPI frontend does not support middlewares. Register your middlewares with the HTTP frontend instead.")
}

func (f *Frontend) Init() apperror.Error {
	apiPrefix := f.registry.Config().UString("api.prefix", "api")

	httpFrontend := f.registry.HttpFrontend()
	if httpFrontend == nil {
		return apperror.New("http_frontend_required", "The JSONAPI frontend relies on the HTTP frontend, which was not found")
	}

	resources := f.registry.Resources()
	for name := range resources {
		name = strings.Replace(name, "_", "-", -1)

		httpFrontend.RegisterHttpHandler("OPTIONS", "/"+apiPrefix+"/"+name, HandleOptions)
		httpFrontend.RegisterHttpHandler("OPTIONS", "/"+apiPrefix+"/"+name+"/:id", HandleOptions)

		// Find.
		httpFrontend.RegisterHttpHandler("GET", "/"+apiPrefix+"/"+name, HandleWrap(name, HandleFind))

		// FindOne.
		httpFrontend.RegisterHttpHandler("GET", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleFindOne))

		// Create.
		httpFrontend.RegisterHttpHandler("POST", "/"+apiPrefix+"/"+name, HandleWrap(name, HandleCreate))

		// Update.
		httpFrontend.RegisterHttpHandler("PATCH", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleUpdate))

		// Delete.
		httpFrontend.RegisterHttpHandler("DELETE", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleDelete))
	}

	return nil
}

func (f *Frontend) Start() apperror.Error {
	// Noop.
	return nil
}

func (f *Frontend) Shutdown() (shutdownChan chan bool, err apperror.Error) {
	return nil, nil
}
