package jsonapi

import (
	"github.com/Sirupsen/logrus"
	"github.com/theduke/go-apperror"

	kit "github.com/theduke/go-appkit"
)

type Frontend struct {
	app   kit.App
	debug bool
}

// Ensure Frontend implements kit.Frontend.
var _ kit.Frontend = (*Frontend)(nil)

func New(app kit.App) *Frontend {
	return &Frontend{
		app: app,
	}
}

func (f *Frontend) Name() string {
	return "jsonapi"
}

func (f *Frontend) App() kit.App {
	return f.app
}

func (f *Frontend) SetApp(x kit.App) {
	f.app = x
}

func (f *Frontend) Debug() bool {
	return f.debug
}

func (f *Frontend) SetDebug(x bool) {
	f.debug = x
}

func (f *Frontend) Logger() *logrus.Logger {
	return f.app.Dependencies().Logger()
}

func (f *Frontend) Init() apperror.Error {
	apiPrefix := f.app.Config().UString("api.prefix", "api")

	resources := f.app.Dependencies().Resources()
	for name := range resources {
		f.app.RegisterHttpHandler("OPTIONS", "/"+apiPrefix+"/"+name, HandleOptions)
		f.app.RegisterHttpHandler("OPTIONS", "/"+apiPrefix+"/"+name+"/:id", HandleOptions)

		// Find.
		f.app.RegisterHttpHandler("GET", "/"+apiPrefix+"/"+name, HandleWrap(name, HandleFind))

		// FindOne.
		f.app.RegisterHttpHandler("GET", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleFindOne))

		// Create.
		f.app.RegisterHttpHandler("POST", "/"+apiPrefix+"/"+name, HandleWrap(name, HandleCreate))

		// Update.
		f.app.RegisterHttpHandler("PATCH", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleUpdate))

		// Delete.
		f.app.RegisterHttpHandler("DELETE", "/"+apiPrefix+"/"+name+"/:id", HandleWrap(name, HandleDelete))
	}

	return nil
}

func (f *Frontend) Start() apperror.Error {
	// Noop.
	return nil
}
