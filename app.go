package appkit

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/manyminds/api2go"
	"github.com/olebedev/config"
	"github.com/spf13/cobra"

	db "github.com/theduke/go-dukedb"

	. "github.com/theduke/go-appkit/error"
	"github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/crawler"

	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/email/gomail"
	emaillog "github.com/theduke/go-appkit/email/log"

	"github.com/theduke/go-appkit/templateengines"
)

type App struct {
	Logger *logrus.Logger

	Debug bool

	ENV string

	Config *config.EnvConfig

	DefaultBackend db.Backend
	backends       map[string]db.Backend

	caches map[string]caches.Cache

	emailService email.EmailService

	templateEngine templateengines.TemplateEngine

	resources   map[string]ApiResource
	userHandler ApiUserHandler
	fileHandler ApiFileHandler

	methods map[string]*Method

	beforeMiddlewares []RequestHandler
	afterMiddlewares  []AfterRequestMiddleware

	serverErrorHandler AfterRequestMiddleware
	notFoundHandler RequestHandler

	sessionManager *SessionManager

	api2go *api2go.API
	router *httprouter.Router

	Cli *cobra.Command
}

func NewApp(cfgPath string) *App {
	app := App{}
	app.resources = make(map[string]ApiResource)
	app.backends = make(map[string]db.Backend)
	app.caches = make(map[string]caches.Cache)
	app.methods = make(map[string]*Method)


	app.beforeMiddlewares = make([]RequestHandler, 0)
	app.afterMiddlewares = make([]AfterRequestMiddleware, 0)

	app.RegisterBeforeMiddleware(RequestTraceMiddleware)
	app.RegisterBeforeMiddleware(AuthenticationMiddleware)

	app.RegisterAfterMiddleware(ServerErrorMiddleware)
	app.RegisterAfterMiddleware(RequestTraceAfterMiddleware)
	app.RegisterAfterMiddleware(RequestLoggerMiddleware)

	// Configure logger.
	app.Logger = &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	app.api2go = api2go.NewAPI("api")
	app.router = app.api2go.Router()

	app.InitCli()
	app.ReadConfig(cfgPath)

	app.RegisterMethod(createMethod())
	app.RegisterMethod(updateMethod())
	app.RegisterMethod(deleteMethod())
	app.RegisterMethod(queryMethod())

	// EmailService setup.
	app.buildEmailService()

	return &app
}

func (a *App) TmpDir() string {
	return a.Config.UString("tmpDir", "tmp")
}

func (a *App) Router() *httprouter.Router {
	return a.router
}

func (a *App) ReadConfig(path string) {
	if path == "" {
		path = "config.yaml"
	}

	var cfg *config.Config

	if f, err := os.Open(path); err != nil {
		a.Logger.Infof("Could not find or read config at '%v' - Using default settings\n", path)
	} else {
		defer f.Close()
		content, err := ioutil.ReadAll(f)
		if err != nil {
			panic(fmt.Sprintf("Could not read config at '%v': %v\n", path, err))
		} else {
			cfg, err = config.ParseYaml(string(content))
			if err != nil {
				panic(fmt.Sprintf("YAML error while parsing config at '%v': %v\n", path, err))
			}
		}
	}

	if cfg == nil {
		defaultConfig := "ENV: dev\ntmpDir: tmp"

		c, _ := config.ParseYaml(defaultConfig)
		cfg = c
	}

	cfg.Env()

	// Set default values if not present.
	env := cfg.UString("ENV", "dev")
	if env == "dev" {
		a.Logger.Info("No environment specified, defaulting to 'dev'")
		a.Debug = true
	}
	a.ENV = env

	// Fill in default values into the config and ensure they are valid.

	// Ensure a tmp directory exists and is readable.
	tmpDir := cfg.UString("tmpDir", "tmp")
	if err := os.MkdirAll(tmpDir, 0777); err != nil {
		panic(fmt.Sprintf("Could not read or create tmp dir at '%v': %v", tmpDir, err))
	}

	a.Config = &config.EnvConfig{Env: env, Config: cfg}
}

func (a *App) PrepareBackends() {
	for name := range a.backends {
		a.backends[name].BuildRelationshipInfo()
	}
}

func (a *App) PrepareForRun() {
	a.PrepareBackends()

	// Auto migrate if enabled or not explicitly disabled and env is debug.
	if auto, err := a.Config.Bool("autoRunMigrations"); (err == nil && auto) || (err != nil && a.ENV == "dev") {
		if err := a.MigrateAllBackends(false); err != nil {
			a.Logger.Errorf("Migration FAILED: %v\n", err)
		}
	}
}

func (a *App) Run() {
	a.PrepareForRun()

	if a.notFoundHandler == nil {
		a.notFoundHandler = notFoundHandler
	}

	// Install not found handler.
	a.router.NotFound = &HttpHandlerStruct{
		App: a,
		Handler: a.notFoundHandler,
	}

	// Install handler for index.
	indexTpl, err := getIndexTpl(a)
	if err != nil {
		panic(err.Error())
	}

	a.router.GET("/", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, func(*App, ApiRequest) (ApiResponse, bool) {
			return &Response{
				RawData: indexTpl,
			}, false
		})
	})

	// Serve files routes.
	serveFiles := a.Config.UMap("serveFiles")
	for route := range serveFiles {
		path, ok := serveFiles[route].(string)
		if !ok {
			a.Logger.Error("Config error: serveFiles configuration invalid: Must be map/dictionary with paths")
			continue
		}
		a.ServeFiles(route, path)
	}

	// Register route for method calls.
	methodHandler := func(a *App, r ApiRequest) (ApiResponse, bool) {
		var response ApiResponse

		responder := func(r ApiResponse) {
			response = r
		}

		method := r.GetContext().String("name")

		finishedChannel, err := a.RunMethod(method, r, responder, true)
		if err != nil {
			return &Response{
				Error: err,
			}, false
		}
		<-finishedChannel

		return response, false
	}

	a.router.POST("/api/method/:name", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, methodHandler)
	})

	// Register api2json resources.
	for key := range a.resources {
		res := a.resources[key]
		a.api2go.AddResource(res.GetModel(), Api2GoResource{
			AppResource: res,
			App:         a,
		})
	}

	// Run the session manager.
	a.sessionManager = NewSessionManager(a)
	a.sessionManager.Run()

	// Crawl on startup if enabled.
	if a.Config.UBool("crawler.onRun", false) {
		a.Crawl()
	}

	url := a.Config.UString("host", "localhost") + ":" + a.Config.UString("port", "8000")
	a.Logger.Infof("Serving on %v", url)

	handler := a.api2go.Handler()
	err2 := http.ListenAndServe(url, handler)
	if err2 != nil {
		a.Logger.Panicf("Could not start server: %v\n", err)
	}
}

/**
 * Crawling.
 */

func (a *App) Crawl() {
	concurrentRequests := a.Config.UInt("crawler.concurrentRequests", 5)
	host := a.Config.UString("url")
	if host == "" {
		a.Logger.Error("Can't crawl because 'url' setting is not specified")
		return
	}

	hosts := []string{host}
	startUrls := []string{"http://" + host + "/"}
	crawler := crawler.New(concurrentRequests, hosts, startUrls)
	crawler.Logger = a.Logger

	go func() {
		crawler.Run()
	}()
}

/**
 * Serve files.
 */

func (a App) ServeFiles(route string, path string) {
	a.Logger.Debugf("Serving files from directory '%v' at route '%v'", path, route)

	server := http.FileServer(http.Dir(path))
	a.Router().GET(route+"/*path", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Fix the url.
		r.URL.Path = params.ByName("path")
		server.ServeHTTP(w, r)
	})
}

/**
 * Backends.
 */

func (a *App) RegisterBackend(name string, b db.Backend) {
	b.SetLogger(a.Logger)
	a.backends[name] = b
	if a.DefaultBackend == nil {
		a.DefaultBackend = b
	}
}

func (a *App) GetBackend(name string) db.Backend {
	b, ok := a.backends[name]
	if !ok {
		panic("Unknown backend: " + name)
	}

	return b
}

/**
 * Caches.
 */

func (a *App) RegisterCache(name string, c caches.Cache) {
	a.caches[name] = c
}

func (a *App) Cache(name string) caches.Cache {
	return a.caches[name]
}

/**
 * Email service.
 */

func (a *App) buildEmailService() {
	host := a.Config.UString("email.host")
	port := a.Config.UInt("email.port")
	user := a.Config.UString("email.user")
	pw := a.Config.UString("email.password")

	fromEmail := a.Config.UString("email.from", "no-reply@appkit")
	fromName := a.Config.UString("email.fromName", "Appkit")

	from := email.Recipient{
		Email: fromEmail,
		Name: fromName,
	}

	if host != "" && port > 0 && user != "" && pw != "" {
		a.emailService = gomail.New(host, port, user, pw, fromEmail, fromName)
		a.Logger.Debug("Using gomail email service")
	} else {
		a.emailService = emaillog.New(a.Logger, from)
		a.Logger.Debug("Using log email service")
	}
}

func (a *App) RegisterEmailService(s email.EmailService) {
	a.emailService = s
}

func (a *App) EmailService() email.EmailService {
	return a.emailService
}

/** 
 * TemplateEngine.
 */

func (a *App) RegisterTemplateEngine(e templateengines.TemplateEngine) {
	a.templateEngine = e
}

func (a *App) TemplateEngine() templateengines.TemplateEngine {
	return a.templateEngine
}

/**
 * Methods.
 */

func (a *App) RegisterMethod(method *Method) {
	if _, exists := a.methods[method.Name]; exists {
		panic(fmt.Sprintf("Method name '%v' already registered.", method.Name))
	}

	a.methods[method.Name] = method
}

func (a *App) RunMethod(name string, r ApiRequest, responder func(ApiResponse), withFinishedChannel bool) (chan bool, Error) {
	if r.GetSession() == nil {
		return nil, AppError{
			Code:    "no_session",
			Message: "Can't run a method without a session",
		}
	}

	method := a.methods[name]
	if method == nil {
		return nil, AppError{
			Code:    "unknown_method",
			Message: fmt.Sprintf("The method %v does not exist", name),
		}
	}

	instance := NewMethodInstance(method, r, responder)

	if withFinishedChannel {
		c := make(chan bool)
		instance.finishedChannel = c
		return c, a.sessionManager.QueueMethod(r.GetSession(), instance)
	} else {
		return nil, a.sessionManager.QueueMethod(r.GetSession(), instance)
	}
}

/**
 * Resources.
 */

func (a *App) RegisterResource(model db.Model, hooks ApiHooks) {
	res := NewResource(model, hooks)
	a.RegisterCustomResource(res)
}

func (a *App) RegisterCustomResource(res ApiResource) {
	res.SetApp(a)
	res.SetDebug(a.Debug)

	if res.GetBackend() == nil {
		if a.DefaultBackend == nil {
			panic("Registering resource without backend, but no default backend set.")
		}

		// Set backend.
		res.SetBackend(a.DefaultBackend)
	}

	// Register model with the backend.
	res.GetBackend().RegisterModel(res.GetModel())

	// Set userhandler if neccessary.
	if res.GetUserHandler() == nil {
		res.SetUserHandler(a.userHandler)
	}

	// Allow a resource to register custom http routes.
	if res.Hooks() != nil {
		if resRoutes, ok := res.Hooks().(ApiHttpRoutes); ok {
			for _, route := range resRoutes.HttpRoutes(res) {
				// Need to wrap this in a lambda, because otherwise, the last routes
				// handler will always be called.
				func(route *HttpRoute) {
					a.Router().Handle(route.Method, route.Route, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
						httpHandler(w, r, params, a, route.Handler)
					})
				}(route)
			}
		}
	}

	a.resources[res.GetModel().Collection()] = res
}

func (a App) GetResource(name string) ApiResource {
	r, ok := a.resources[name]
	if !ok {
		return nil
	}

	return r
}

/**
 * UserHandler.
 */

func (a *App) RegisterUserHandler(h ApiUserHandler) {
	a.userHandler = h
	a.RegisterCustomResource(h.GetUserResource())
	a.RegisterCustomResource(h.GetSessionResource())

	if profileModel := h.GetProfileModel(); profileModel != nil {
		a.DefaultBackend.RegisterModel(profileModel)
	}

	auth := h.GetAuthItemResource()
	if auth.GetBackend() == nil {
		a.DefaultBackend.RegisterModel(auth.GetModel())
		auth.SetBackend(a.DefaultBackend)
	}

	roles := h.GetRoleResource()
	if roles.GetBackend() == nil {
		a.DefaultBackend.RegisterModel(roles.GetModel())
		roles.SetBackend(a.DefaultBackend)
	}

	permissions := h.GetPermissionResource()
	if permissions.GetBackend() == nil {
		a.DefaultBackend.RegisterModel(permissions.GetModel())
		permissions.SetBackend(a.DefaultBackend)
	}
}

/**
 * FileHandler.
 */

func (a *App) RegisterFileHandler(f ApiFileHandler) {
	r := f.Resource()
	if r == nil {
		panic("Trying to register file handler without resource")
	}

	a.RegisterCustomResource(r)
	f.SetApp(a)

	a.fileHandler = f
}

func (a *App) FileHandler() ApiFileHandler {
	return a.fileHandler
}

/**
 * Middlewares.
 */

func (a *App) RegisterBeforeMiddleware(handler RequestHandler) {
	a.beforeMiddlewares = append(a.beforeMiddlewares, handler)
}

func (a *App) ClearBeforeMiddleware() {
	a.beforeMiddlewares = a.beforeMiddlewares[:]
}

func (a *App) GetBeforeMiddlewares() []RequestHandler {
	return a.beforeMiddlewares
}

func (a *App) RegisterAfterMiddleware(middleware AfterRequestMiddleware) {
	a.afterMiddlewares = append(a.afterMiddlewares, middleware)
}

func (a *App) ClearAfterMiddlewares() {
	a.afterMiddlewares = a.afterMiddlewares[:]
}

func (a *App) GetAfterMiddlewares() []AfterRequestMiddleware {
	return a.afterMiddlewares
}

/**
 * Http handlers.
 */

func(a *App) NotFoundHandler() RequestHandler {
	return a.notFoundHandler
}

func(a *App) SetNotFoundHandler(x RequestHandler) {
	a.notFoundHandler = x
}

func (a *App) RegisterHttpHandler(method, path string, handler RequestHandler) {
	a.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, handler)
	})
}

/**
 * Migrations and Backend functionality.
 */

func (a *App) MigrateBackend(name string, version int, force bool) Error {
	a.Logger.Infof("MIGRATE: Migrating backend '%v'", name)
	backend := a.GetBackend(name)
	if backend == nil {
		return AppError{
			Code:    "unknown_backend",
			Message: fmt.Sprint("The backend '%v' does not exist", name),
		}
	}

	migrationBackend, ok := backend.(db.MigrationBackend)
	if !ok {
		return AppError{
			Code:    "backend_cant_migrate",
			Message: fmt.Sprintf("The backend '%v' does not support migrations", name),
		}
	}

	if version == 0 {
		return migrationBackend.GetMigrationHandler().Migrate(force)
	} else {
		return migrationBackend.GetMigrationHandler().MigrateTo(version, force)
	}

	return nil
}

func (a *App) MigrateAllBackends(force bool) Error {
	a.Logger.Infof("MIGRATE: Migrating all backends to newest version")
	for key := range a.backends {
		if err := a.MigrateBackend(key, 0, force); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) DropBackend(name string) Error {
	b := a.GetBackend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	a.Logger.Infof("Dropping all collections on backend " + name)

	if err := b.DropAllCollections(); err != nil {
		a.Logger.Errorf("Dropping all collections failed: %v", err)
		return err
	}

	return nil
}

func (a *App) DropAllBackends() Error {
	a.Logger.Infof("Dropping all backends")
	for name := range a.backends {
		if err := a.DropBackend(name); err != nil {
			return err
		}
	}
	a.Logger.Infof("Successfully dropped all collections")
	return nil
}

func (a *App) RebuildBackend(name string) Error {
	b := a.GetBackend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	a.Logger.Infof("Rebuilding backend " + name)

	if err := a.DropBackend(name); err != nil {
		return err
	}

	if err := a.MigrateBackend(name, 0, false); err != nil {
		a.Logger.Errorf("Migration failed: %v", err)
		return AppError{
			Code:    "backend_migration_failed",
			Message: err.Error(),
		}
	}

	return nil
}

func (a *App) RebuildAllBackends() Error {
	a.Logger.Infof("Rebuilding all backends")
	for key := range a.backends {
		if err := a.RebuildBackend(key); err != nil {
			return err
		}
	}

	a.Logger.Infof("Successfully migrated all backends")

	return nil
}

func (a App) GetUserHandler() ApiUserHandler {
	return a.userHandler
}
