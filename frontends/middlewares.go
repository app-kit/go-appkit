package frontends

import (
	"time"

	"github.com/Sirupsen/logrus"

	kit "github.com/theduke/go-appkit"
)

/**
 * Request trace middlewares.
 */

func RequestTraceMiddleware(registry kit.Registry, r kit.Request) (kit.Response, bool) {
	r.GetContext().Set("startTime", time.Now())
	return nil, false
}

func RequestTraceAfterMiddleware(registry kit.Registry, r kit.Request, response kit.Response) (kit.Response, bool) {
	r.GetContext().Set("endTime", time.Now())
	return nil, false
}

func SerializeResponseMiddleware(registry kit.Registry, request kit.Request, response kit.Response) (kit.Response, bool) {
	// Try to serialize the reponse data.

	// Determine serializer.
	serializer := registry.DefaultSerializer()

	// Check if a custom serializer was specified.
	if name := request.GetContext().String("response-serializer"); name != "" {
		serializer = registry.Serializer(name)
		if serializer == nil {
			errResp := kit.NewErrorResponse("unknown_response_serializer", true)
			data := serializer.MustSerializeResponse(errResp)
			errResp.SetData(data)
			return errResp, false
		}
	}

	// Set format in metadata.
	meta := response.GetMeta()
	if meta == nil {
		meta = make(map[string]interface{})
	}

	meta["format"] = serializer.Name()
	response.SetMeta(meta)

	data := serializer.MustSerializeResponse(response)
	response.SetData(data)

	return nil, false
}

func RequestLoggerMiddleware(registry kit.Registry, r kit.Request, response kit.Response) (kit.Response, bool) {

	// Calculate time taken.
	rawStarted, ok1 := r.GetContext().Get("startTime")
	rawFinished, ok2 := r.GetContext().Get("endTime")

	timeTaken := int64(-1)
	if ok1 && ok2 {
		started := rawStarted.(time.Time)
		finished := rawFinished.(time.Time)
		timeTaken = int64(finished.Sub(started) / time.Millisecond)
	}

	// Log the request.
	method := r.GetHttpMethod()
	path := r.GetPath()
	if response.GetError() != nil {
		registry.Logger().WithFields(logrus.Fields{
			"frontend":     r.GetFrontend(),
			"action":       "request",
			"method":       method,
			"path":         path,
			"status":       response.GetHttpStatus(),
			"err":          response.GetError(),
			"milliseconds": timeTaken,
		}).Errorf("%v: %v - %v - %v", response.GetHttpStatus(), method, path, response.GetError())
	} else {
		registry.Logger().WithFields(logrus.Fields{
			"frontend":     r.GetFrontend(),
			"action":       "request",
			"method":       method,
			"path":         path,
			"status":       response.GetHttpStatus(),
			"milliseconds": timeTaken,
		}).Debugf("%v: %v - %v", response.GetHttpStatus(), method, path)
	}

	return nil, false
}
