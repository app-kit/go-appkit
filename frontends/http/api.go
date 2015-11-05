package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/caches"
	"github.com/app-kit/go-appkit/utils"
)

type HttpHandlerStruct struct {
	Registry kit.Registry
	Handler  kit.RequestHandler
}

func (h *HttpHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := new(httprouter.Params)
	HttpHandler(w, r, *params, h.Registry, h.Handler)
}

func serverRenderer(registry kit.Registry, r kit.Request) kit.Response {
	url := r.GetHttpRequest().URL

	// Build the url to query.
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	if url.Host == "" {
		url.Host = registry.Config().UString("host", "localhost") + ":" + registry.Config().UString("port", "8000")
	}

	q := url.Query()
	q.Set("no-server-render", "1")
	url.RawQuery = q.Encode()

	strUrl := url.String()

	cacheKey := "serverrenderer_" + strUrl
	cacheName := registry.Config().UString("serverRenderer.cache")
	var cache kit.Cache

	// If a cache is specified, try to retrieve it.
	if cacheName != "" {
		cache = registry.Cache(cacheName)
		if cache == nil {
			registry.Logger().Errorf("serverRenderer.cache is set to %v, but the cache is not registered with app", cacheName)
		}
	}

	// If a cache was found, try to retrieve cached response.
	if cache != nil {
		item, err := cache.Get(cacheKey)
		if err != nil {
			registry.Logger().Errorf("serverRenderer: cache retrieval error: %v", err)
		} else if item != nil {
			// Cache item found, return response with cache item.
			status, _ := strconv.ParseInt(item.GetTags()[0], 10, 64)
			data, _ := item.ToString()

			return &kit.AppResponse{
				HttpStatus: int(status),
				RawData:    []byte(data),
			}
		}
	}

	// Either no cache or url not yet cached, so render it.

	// First, ensure that the tmp directory exists.
	tmpDir := path.Join(registry.Config().TmpDir(), "phantom")
	if ok, _ := utils.FileExists(tmpDir); !ok {
		if err := os.MkdirAll(tmpDir, 0777); err != nil {
			return &kit.AppResponse{
				Error: &apperror.Err{
					Code:    "create_tmp_dir_failed",
					Message: fmt.Sprintf("Could not create the tmp directory at %v: %v", tmpDir, err),
				},
			}
		}
	}

	// Build a unique file name.
	filePath := path.Join(tmpDir, utils.UUIdv4()+".html")

	// Execute phantom js.

	// Find path of phantom script.
	_, filename, _, _ := runtime.Caller(1)
	scriptPath := path.Join(path.Dir(path.Dir(filename)), "phantom", "render.js")

	start := time.Now()

	phantomPath := registry.Config().UString("serverRenderer.phantomJsPath", "phantomjs")

	args := []string{
		"--web-security=false",
		"--local-to-remote-url-access=true",
		scriptPath,
		"10",
		strUrl,
		filePath,
	}
	result, err := exec.Command(phantomPath, args...).CombinedOutput()
	if err != nil {
		registry.Logger().Errorf("Phantomjs execution error: %v", string(result))

		return &kit.AppResponse{
			Error: apperror.Wrap(err, "phantom_execution_failed"),
		}
	}

	// Get time taken as milliseconds.
	timeTaken := int(time.Now().Sub(start) / time.Millisecond)
	registry.Logger().WithFields(log.Fields{
		"action":       "phantomjs_render",
		"milliseconds": timeTaken,
	}).Debugf("Rendered url %v with phantomjs", url)

	content, err2 := utils.ReadFile(filePath)
	if err2 != nil {
		return kit.NewErrorResponse(err2)
	}

	// Find http status code.
	status := 200
	res := regexp.MustCompile("http_status_code\\=(\\d+)").FindStringSubmatch(string(content))
	if res != nil {
		s, _ := strconv.ParseInt(res[1], 10, 64)
		status = int(s)
	}

	// Save to cache.
	if cache != nil {
		lifetime := registry.Config().UInt("serverRenderer.cacheLiftetime", 3600)

		err := cache.Set(&caches.StrItem{
			Key:       cacheKey,
			Value:     string(content),
			Tags:      []string{strconv.FormatInt(int64(status), 10)},
			ExpiresAt: time.Now().Add(time.Duration(lifetime) * time.Second),
		})
		if err != nil {
			registry.Logger().Errorf("serverRenderer: Cache persist error: %v", err)
		}
	}

	return &kit.AppResponse{
		HttpStatus: status,
		RawData:    content,
	}
}

func defaultErrorTpl() *template.Template {
	tpl := `
	<html>
		<head>
			<title>Server Error</title>
		</head>

		<body>
			<h1>Server Error</h1>

			<p>{{.error}}</p>
		</body>
	</html>
	`

	t := template.Must(template.New("error").Parse(tpl))
	return t
}

func defaultNotFoundTpl() *template.Template {
	tpl := `
	<html>
		<head>
			<title>Page Not Found</title>
		</head>

		<body>
			<h1>Page Not Found</h1>

			<p>The page you are looking for does not exist.</p>
		</body>
	</html>
	`

	t, _ := template.New("error").Parse(tpl)
	return t
}

func getIndexTpl(registry kit.Registry) ([]byte, apperror.Error) {
	if path := registry.Config().UString("frontend.indexTpl"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, apperror.Wrap(err, "index_tpl_open_error",
				fmt.Sprintf("The index template at %v could not be opened", path))
		}

		tpl, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, apperror.Wrap(err, "index_tpl_read_error",
				fmt.Sprintf("Could not read index template at %v", path))
		}

		return tpl, nil
	}

	tpl := `
	<html>
		<body>
			<h1>Go Appkit</h1>

			<p>Welcome to your new appkit server.</p>

			<p>
			  Find instructions on how to set up your app at <a href="http://github.com/app-kit/go-appkit">Github</a>
			</p>
		</body>
	</html>
	`

	return []byte(tpl), nil
}

func notFoundHandler(registry kit.Registry, r kit.Request) (kit.Response, bool) {
	httpRequest := r.GetHttpRequest()
	apiPrefix := "/" + registry.Config().UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	// Try to render the page on the server, if enabled.
	if !isApiRequest {
		renderEnabled := registry.Config().UBool("serverRenderer.enabled", false)
		noRender := strings.Contains(httpRequest.URL.String(), "no-server-render")

		if renderEnabled && !noRender {
			return serverRenderer(registry, r), false
		}
	}

	// For non-api requests, render the default template.
	if !isApiRequest {
		tpl, err := getIndexTpl(registry)
		if err != nil {
			return kit.NewErrorResponse(err), false
		}
		return &kit.AppResponse{
			RawData: tpl,
		}, false
	}

	// For api requests, render the api not found error.
	return &kit.AppResponse{
		Error: &apperror.Err{
			Code:    "not_found",
			Message: "This api route does not exist",
		},
	}, false
}

func RespondWithContent(app kit.App, w http.ResponseWriter, code int, content []byte) {
	w.WriteHeader(code)
	w.Write(content)
}

func RespondWithReader(app kit.App, w http.ResponseWriter, code int, reader io.ReadCloser) {
	w.WriteHeader(code)
	io.Copy(w, reader)
	reader.Close()
}

func RespondWithJson(w http.ResponseWriter, response kit.Response) {
	code := 200
	respData := map[string]interface{}{
		"data": response.GetData(),
	}

	if response.GetError() != nil {
		errs := []error{response.GetError()}

		additionalErrs := response.GetError().GetErrors()
		if additionalErrs != nil {
			for _, err := range additionalErrs {
				if apiErr, ok := err.(apperror.Error); ok && apiErr.IsPublic() {
					errs = append(errs, apiErr)
				}
			}
		}

		respData["errors"] = errs
		code = 500
	}

	output, err2 := json.Marshal(respData)
	if err2 != nil {
		code = 500
		respData = map[string]interface{}{
			"errors": []error{
				apperror.Wrap(err2, "json_encode_error"),
			},
		}
		output, _ = json.Marshal(respData)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(output)
}

func processRequest(registry kit.Registry, request kit.Request, handler kit.RequestHandler) (kit.Response, bool) {
	var response kit.Response

	// Run before middlewares.
	for _, middleware := range registry.HttpFrontend().BeforeMiddlewares() {
		var skip bool
		response, skip = middleware(registry, request)
		if skip {
			return nil, true
		} else if response != nil {
			break
		}
	}

	// Only run the handler if no middleware provided a response.
	if response == nil {
		skip := false
		response, skip = handler(registry, request)
		if skip {
			return nil, true
		}
	}

	if response.GetHttpStatus() == 0 {
		// Note: error handler middleware will set proper http status for error responses.
		response.SetHttpStatus(200)
	}

	// Run after request middlewares.
	// Note: error responses are converted with the serverErrrorMiddleware middleware.
	// Note: serializing the response into the proper format is done with the SerializeResponseMiddleware.
	for _, middleware := range registry.HttpFrontend().AfterMiddlewares() {
		resp, skip := middleware(registry, request, response)
		if skip {
			return nil, true
		} else if resp != nil {
			response = resp
		}
	}

	return response, false
}

func HttpHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params, registry kit.Registry, handler kit.RequestHandler) {
	config := registry.Config()
	header := w.Header()

	// Set Access-Control headers.
	allowedOrigins := config.UString("accessControl.allowedOrigins", "*")
	header.Set("Access-Control-Allow-Origin", allowedOrigins)

	methods := config.UString("accessControl.allowedMethods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	header.Set("Access-Control-Allow-Methods", methods)

	allowedHeaders := config.UString("accessControl.allowedHeaders", "Authentication, Content-Type, X-Requested-With, Accept, Accept-Language, Content-Language")
	header.Set("Access-Control-Allow-Headers", allowedHeaders)

	// If it is an options request, just respond with 200.
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	request := kit.NewRequest()
	request.SetFrontend("http")
	request.SetPath(r.URL.String())
	request.SetHttpMethod(r.Method)
	request.SetHttpRequest(r)
	request.SetHttpResponseWriter(w)

	for _, param := range params {
		request.Context.Set(param.Key, param.Value)
	}

	queryVals := r.URL.Query()
	for key := range queryVals {
		vals := queryVals[key]
		if len(vals) == 1 {
			request.Context.Set(key, vals[0])
		} else {
			request.Context.Set(key, vals)
		}
	}

	response, skip := processRequest(registry, request, handler)
	if skip {
		return
	}

	// If a data reader is set, write the data of the reader.
	reader := response.GetRawDataReader()
	if reader != nil {
		w.WriteHeader(response.GetHttpStatus())
		io.Copy(w, reader)
		reader.Close()
		return
	}

	// If raw data is set, write the raw data.
	rawData := response.GetRawData()
	if rawData != nil {
		w.WriteHeader(response.GetHttpStatus())
		w.Write(rawData)
		return
	}

	registry.Logger().Panicf("Invalid response with no raw data: %+v", response)
}

/**
 * Before middlewares.
 */

func UnserializeRequestMiddleware(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	// Try to parse json in body. Ignore error since body might not contain json.
	contentType := request.GetHttpRequest().Header.Get("Content-Type")
	if strings.Contains(contentType, "json") {
		// Only read the HTTP body automatically for json content type requests,
		// since some handlers might need to read it themselfes (see the files package resource).
		if err := request.ReadHttpBody(); err != nil {
			return kit.NewErrorResponse(err, "http_body_read_error"), false
		} else {
			if request.GetRawData() != nil {
				if err := request.ParseJsonData(); err != nil {
					return kit.NewErrorResponse(err, "invalid_json_body", true), false
				}

				if request.GetData() != nil {
					// Successfully parsed json body.

					// Now try to unserialize.

					// Determine serializer.
					serializer := registry.DefaultSerializer()

					// Check if a custom serializer was specified.
					if name := request.GetContext().String("request-serializer"); name != "" {
						serializer = registry.Serializer(name)
					}

					if serializer == nil {
						return kit.NewErrorResponse("unknown_serializer", fmt.Sprintf("The specified request serializer does not exist")), false
					} else {
						if err := request.Unserialize(serializer); err != nil {
							return kit.NewErrorResponse(err, "request_unserialize_error", true), false
						}
					}
				}
			}
		}
	}

	return nil, false
}

func AuthenticationMiddleware(registry kit.Registry, r kit.Request) (kit.Response, bool) {
	// Handle authentication.
	httpRequest := r.GetHttpRequest()
	userService := registry.UserService()

	if userService == nil {
		return nil, false
	}

	authHeader := httpRequest.Header.Get("Authentication")
	if authHeader == "" {
		return nil, false
	}

	// Check for basic auth.
	if strings.HasPrefix(authHeader, "Basic ") {
		str := authHeader[6:]
		data, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return kit.NewErrorResponse("invalid_basic_auth"), false
		} else {
			parts := strings.Split(string(data), ":")
			if len(parts) == 2 {
				userIdentifier := parts[0]
				pw := parts[1]

				user, err := userService.AuthenticateUser(userIdentifier, "password", map[string]interface{}{"password": pw})
				if err != nil {
					return kit.NewErrorResponse(err), false
				}

				r.SetUser(user)
				return nil, false
			}
		}
	}

	// Check for auth token.
	if authHeader != "" {
		token := authHeader
		user, session, err := userService.VerifySession(token)
		if err == nil {
			r.SetUser(user)
			r.SetSession(session)

			return nil, false
		} else {
			return kit.NewErrorResponse(err), false
		}
	}

	return nil, false
}

/**
 * After middlewares.
 */

func ServerErrorMiddleware(registry kit.Registry, r kit.Request, response kit.Response) (kit.Response, bool) {
	err := response.GetError()
	if err == nil {
		return nil, false
	}

	status := 500

	// If the error is an apperror, and it contains a status,
	// set it as the http status of the response.
	if apperr, ok := err.(apperror.Error); ok {
		if apperr.GetStatus() != 0 {
			status = apperr.GetStatus()
		}
	}

	response.SetHttpStatus(status)

	if response.GetRawData() != nil || response.GetRawDataReader() != nil {
		return nil, false
	}

	httpRequest := r.GetHttpRequest()
	apiPrefix := "/" + registry.Config().UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	if isApiRequest {
		return nil, false
	}

	data := map[string]interface{}{"errors": []error{response.GetError()}}

	tpl := defaultErrorTpl()

	tplPath := registry.Config().UString("frontend.errorTemplate")
	if tplPath != "" {
		t, err := template.ParseFiles(tplPath)
		if err != nil {
			registry.Logger().Fatalf("Could not parse error template at '%v': %v", tplPath, err)
		} else {
			tpl = t
		}
	}

	var buffer *bytes.Buffer
	if err := tpl.Execute(buffer, data); err != nil {
		registry.Logger().Fatalf("Could not render error template: %v\n", err)
		response.SetRawData([]byte("Server error"))
	} else {
		response.SetRawData(buffer.Bytes())
	}

	return nil, false
}

func MarshalResponseMiddleware(registry kit.Registry, request kit.Request, response kit.Response) (kit.Response, bool) {
	js, err := json.Marshal(response.GetData())
	if err != nil {
		js = []byte(`{"errors": [{code: "response_marshal_error"}]}`)
	}

	response.SetRawData(js)

	return nil, false
}
