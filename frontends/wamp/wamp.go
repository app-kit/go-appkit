package wamp

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/theduke/go-apperror"
	"gopkg.in/jcelliott/turnpike.v2"

	kit "github.com/theduke/go-appkit"
)

type Frontend struct {
	registry kit.Registry
	debug    bool

	server *turnpike.WebsocketServer
	client *turnpike.Client
}

// Ensure that Frontend implements appkit.Frontend.
var _ kit.Frontend = (*Frontend)(nil)

func New(registry kit.Registry) *Frontend {
	f := &Frontend{
		registry: registry,
		debug:    true,
	}

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

	// Register websocket handler.
	httpFrontend.Router().Handle("GET", path, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		fmt.Printf("Handling wamp request")
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

		request := &kit.AppRequest{
			Data: kwargs["data"],
		}

		rawMeta := kwargs["meta"]
		if rawMeta != nil {
			meta, _ := rawMeta.(map[string]interface{})
			request.Meta = kit.NewContext(meta)
		}

		var response kit.Response

		responder := func(r kit.Response) {
			response = r
		}

		finishedChannel, err := f.registry.App().RunMethod(methodName, request, responder, true)
		if err != nil {
			return convertResponse(kit.NewErrorResponse(err))
		}
		<-finishedChannel

		return convertResponse(response)
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
