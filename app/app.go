package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/manyminds/api2go"
	"github.com/olebedev/config"
	"github.com/spf13/cobra"

	db "github.com/theduke/go-dukedb"

	. "github.com/theduke/go-appkit/error"
	kit "github.com/theduke/go-appkit"
	//"github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/crawler"
	"github.com/theduke/go-appkit/resources"

	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/email/gomail"
	emaillog "github.com/theduke/go-appkit/email/log"
)

type App struct {
	env string
	debug bool

	logger *logrus.Logger

	config *config.EnvConfig

	defaultBackend db.Backend
	backends       map[string]db.Backend

	caches map[string]kit.Cache

	emailService kit.EmailService

	templateEngine kit.TemplateEngine

	resources   map[string]kit.Resource
	userService kit.UserService
	fileService kit.FileService

	methods map[string]kit.Method

	beforeMiddlewares []kit.RequestHandler
	afterMiddlewares  []kit.AfterRequestMiddleware

	serverErrorHandler kit.AfterRequestMiddleware
	notFoundHandler kit.RequestHandler

	sessionManager *SessionManager

	api2go *api2go.API
	router *httprouter.Router

	Cli *cobra.Command
}

// Ensure App implements App interface.
var _ kit.App = (*App)(nil)

func NewApp(cfgPath string) *App {
	app := App{}
	app.resources = make(map[string]kit.Resource)
	app.backends = make(map[string]db.Backend)
	app.caches = make(map[string]kit.Cache)
	app.methods = make(map[string]kit.Method)


	app.beforeMiddlewares = make([]kit.RequestHandler, 0)
	app.afterMiddlewares = make([]kit.AfterRequestMiddleware, 0)

	app.RegisterBeforeMiddleware(RequestTraceMiddleware)
	app.RegisterBeforeMiddleware(AuthenticationMiddleware)

	app.RegisterAfterMiddleware(ServerErrorMiddleware)
	app.RegisterAfterMiddleware(RequestTraceAfterMiddleware)
	app.RegisterAfterMiddleware(RequestLoggerMiddleware)

	// Configure logger.
	app.logger = &logrus.Logger{
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

func(a *App) ENV() string {
	return a.env
}

func(a *App) SetENV(x string) {
	a.env = x
}

func(a *App) Debug() bool {
	return a.debug
}

func(a *App) SetDebug(x bool) {
	a.debug = x
}

func(a *App) Logger() *logrus.Logger {
	return a.logger
}

func(a *App) SetLogger(x *logrus.Logger) {
	a.logger = x
}


func (a *App) Router() *httprouter.Router {
	return a.router
}

func (a *App) TmpDir() string {
	return a.config.UString("tmpDir", "tmp")
}

/**
 * Config.
 */

func(a *App) Config() *config.EnvConfig {
	return a.config
}

func(a *App) SetConfig(x *config.EnvConfig) {
	a.config = x
}

func (a *App) ReadConfig(path string) {
	if path == "" {
		path = "config.yaml"
	}

	var cfg *config.Config

	if f, err := os.Open(path); err != nil {
		a.logger.Infof("Could not find or read config at '%v' - Using default settings\n", path)
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
		a.logger.Info("No environment specified, defaulting to 'dev'")
		a.debug = true
	}
	a.env = env

	// Fill in default values into the config and ensure they are valid.

	// Ensure a tmp directory exists and is readable.
	tmpDir := cfg.UString("tmpDir", "tmp")
	if err := os.MkdirAll(tmpDir, 0777); err != nil {
		panic(fmt.Sprintf("Could not read or create tmp dir at '%v': %v", tmpDir, err))
	}

	a.config = &config.EnvConfig{Env: env, Config: cfg}
}

func (a *App) PrepareBackends() {
	for name := range a.backends {
		a.backends[name].BuildRelationshipInfo()
	}
}

func (a *App) PrepareForRun() {
	a.PrepareBackends()

	// Auto migrate if enabled or not explicitly disabled and env is debug.
	if auto, err := a.config.Bool("autoRunMigrations"); (err == nil && auto) || (err != nil && a.env == "dev") {
		if err := a.MigrateAllBackends(false); err != nil {
			a.logger.Errorf("Migration FAILED: %v\n", err)
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
		httpHandler(w, r, params, a, func(kit.App, kit.Request) (kit.Response, bool) {
			return &kit.AppResponse{
				RawData: indexTpl,
			}, false
		})
	})

	// Serve files routes.
	serveFiles := a.config.UMap("serveFiles")
	for route := range serveFiles {
		path, ok := serveFiles[route].(string)
		if !ok {
			a.logger.Error("Config error: serveFiles configuration invalid: Must be map/dictionary with paths")
			continue
		}
		a.ServeFiles(route, path)
	}

	// Register route for method calls.
	methodHandler := func(a kit.App, r kit.Request) (kit.Response, bool) {
		var response kit.Response

		responder := func(r kit.Response) {
			response = r
		}

		method := r.GetContext().String("name")

		finishedChannel, err := a.RunMethod(method, r, responder, true)
		if err != nil {
			return &kit.AppResponse{
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
		a.api2go.AddResource(res.Model(), Api2GoResource{
			AppResource: res,
			App:         a,
		})
	}

	// Run the session manager.
	a.sessionManager = NewSessionManager(a)
	a.sessionManager.Run()

	// Crawl on startup if enabled.
	if a.config.UBool("crawler.onRun", false) {
		a.RunCrawler()
	}

	url := a.config.UString("host", "localhost") + ":" + a.config.UString("port", "8000")
	a.logger.Infof("Serving on %v", url)

	handler := a.api2go.Handler()
	err2 := http.ListenAndServe(url, handler)
	if err2 != nil {
		a.logger.Panicf("Could not start server: %v\n", err)
	}
}

/**
 * Crawling.
 */

func (a *App) RunCrawler() {
	recrawlInterval := a.config.UInt("crawler.recrawlInterval", 0)

	a.logger.Infof("Running crawler with recrawl interval %v", recrawlInterval)

	go func() {
		for {
			a.Crawl()
			if recrawlInterval == 0 {
				break
			} else {
				time.Sleep(time.Second * time.Duration(recrawlInterval))
				a.Crawl()
			}
		}
	}()
}

func (a *App) Crawl() {
	concurrentRequests := a.config.UInt("crawler.concurrentRequests", 5)
	host := a.config.UString("url")
	if host == "" {
		a.logger.Error("Can't crawl because 'url' setting is not specified")
		return
	}

	hosts := []string{host}
	startUrls := []string{"http://" + host + "/"}
	crawler := crawler.New(concurrentRequests, hosts, startUrls)
	crawler.Logger = a.logger

	crawler.Run()
}

/**
 * Serve files.
 */

func (a App) ServeFiles(route string, path string) {
	a.logger.Debugf("Serving files from directory '%v' at route '%v'", path, route)

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
	b.SetLogger(a.logger)
	a.backends[name] = b
	if a.defaultBackend == nil {
		a.defaultBackend = b
	}
}

func (a *App) Backend(name string) db.Backend {
	b, ok := a.backends[name]
	if !ok {
		panic("Unknown backend: " + name)
	}

	return b
}

/**
 * Caches.
 */

func (a *App) RegisterCache(name string, c kit.Cache) {
	a.caches[name] = c
}

func (a *App) Cache(name string) kit.Cache {
	return a.caches[name]
}

/**
 * Email service.
 */

func (a *App) buildEmailService() {
	host := a.config.UString("email.host")
	port := a.config.UInt("email.port")
	user := a.config.UString("email.user")
	pw := a.config.UString("email.password")

	fromEmail := a.config.UString("email.from", "no-reply@appkit")
	fromName := a.config.UString("email.fromName", "Appkit")

	from := email.Recipient{
		Email: fromEmail,
		Name: fromName,
	}

	if host != "" && port > 0 && user != "" && pw != "" {
		a.emailService = gomail.New(host, port, user, pw, fromEmail, fromName)
		a.logger.Debug("Using gomail email service")
	} else {
		a.emailService = emaillog.New(a.logger, from)
		a.logger.Debug("Using log email service")
	}
}

func (a *App) RegisterEmailService(s kit.EmailService) {
	a.emailService = s
}

func (a *App) EmailService() kit.EmailService {
	return a.emailService
}

/** 
 * TemplateEngine.
 */

func (a *App) RegisterTemplateEngine(e kit.TemplateEngine) {
	a.templateEngine = e
}

func (a *App) TemplateEngine() kit.TemplateEngine {
	return a.templateEngine
}

/**
 * Methods.
 */

func (a *App) RegisterMethod(method kit.Method) {
	if _, exists := a.methods[method.Name()]; exists {
		panic(fmt.Sprintf("Method name '%v' already registered.", method.Name()))
	}

	a.methods[method.Name()] = method
}

func (a *App) RunMethod(name string, r kit.Request, responder func(kit.Response), withFinishedChannel bool) (chan bool, Error) {
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

func (a *App) RegisterResource(model db.Model, hooks interface{}) {
	res := resources.NewResource(model, hooks)
	a.RegisterCustomResource(res)
}

func (a *App) RegisterCustomResource(res kit.Resource) {
	res.SetApp(a)
	res.SetDebug(a.debug)

	if res.Backend() == nil {
		if a.defaultBackend == nil {
			panic("Registering resource without backend, but no default backend set.")
		}

		// Set backend.
		res.SetBackend(a.defaultBackend)
	}

	// Register model with the backend.
	res.Backend().RegisterModel(res.Model())

	// Set userhandler if neccessary.
	if res.UserService() == nil {
		res.SetUserService(a.userService)
	}

	// Allow a resource to register custom http routes.
	if res.Hooks() != nil {
		if resRoutes, ok := res.Hooks().(resources.ApiHttpRoutes); ok {
			for _, route := range resRoutes.HttpRoutes(res) {
				// Need to wrap this in a lambda, because otherwise, the last routes
				// handler will always be called.
				func(route kit.HttpRoute) {
					a.Router().Handle(route.Method(), route.Route(), func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
						httpHandler(w, r, params, a, route.Handler())
					})
				}(route)
			}
		}
	}

	a.resources[res.Model().Collection()] = res
}

func (a App) Resource(name string) kit.Resource {
	r, ok := a.resources[name]
	if !ok {
		return nil
	}

	return r
}

/**
 * UserHandler.
 */

func (a *App) RegisterUserService(h kit.UserService) {
	a.userService = h
	a.RegisterCustomResource(h.UserResource())
	a.RegisterCustomResource(h.SessionResource())

	if profileModel := h.ProfileModel(); profileModel != nil {
		a.defaultBackend.RegisterModel(profileModel)
	}

	auth := h.AuthItemResource()
	if auth.Backend() == nil {
		a.defaultBackend.RegisterModel(auth.Model())
		auth.SetBackend(a.defaultBackend)
	}

	roles := h.RoleResource()
	if roles.Backend() == nil {
		a.defaultBackend.RegisterModel(roles.Model())
		roles.SetBackend(a.defaultBackend)
	}

	permissions := h.PermissionResource()
	if permissions.Backend() == nil {
		a.defaultBackend.RegisterModel(permissions.Model())
		permissions.SetBackend(a.defaultBackend)
	}
}

func (a *App) UserService() kit.UserService {
	return a.userService
}

/**
 * FileService.
 */

func (a *App) RegisterFileService(f kit.FileService) {
	r := f.Resource()
	if r == nil {
		panic("Trying to register file handler without resource")
	}

	a.RegisterCustomResource(r)
	f.SetApp(a)

	a.fileService = f
}

func (a *App) FileService() kit.FileService {
	return a.fileService
}

/**
 * Middlewares.
 */

func (a *App) RegisterBeforeMiddleware(handler kit.RequestHandler) {
	a.beforeMiddlewares = append(a.beforeMiddlewares, handler)
}

func (a *App) ClearBeforeMiddlewares() {
	a.beforeMiddlewares = a.beforeMiddlewares[:]
}

func (a *App) BeforeMiddlewares() []kit.RequestHandler {
	return a.beforeMiddlewares
}

func (a *App) RegisterAfterMiddleware(middleware kit.AfterRequestMiddleware) {
	a.afterMiddlewares = append(a.afterMiddlewares, middleware)
}

func (a *App) ClearAfterMiddlewares() {
	a.afterMiddlewares = a.afterMiddlewares[:]
}

func (a *App) AfterMiddlewares() []kit.AfterRequestMiddleware {
	return a.afterMiddlewares
}

/**
 * Http handlers.
 */

func(a *App) NotFoundHandler() kit.RequestHandler {
	return a.notFoundHandler
}

func(a *App) SetNotFoundHandler(x kit.RequestHandler) {
	a.notFoundHandler = x
}

func (a *App) RegisterHttpHandler(method, path string, handler kit.RequestHandler) {
	a.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, handler)
	})
}

/**
 * Migrations and Backend functionality.
 */

func (a *App) MigrateBackend(name string, version int, force bool) Error {
	a.logger.Infof("MIGRATE: Migrating backend '%v'", name)
	backend := a.Backend(name)
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
	a.logger.Infof("MIGRATE: Migrating all backends to newest version")
	for key := range a.backends {
		if err := a.MigrateBackend(key, 0, force); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) DropBackend(name string) Error {
	b := a.Backend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	a.logger.Infof("Dropping all collections on backend " + name)

	if err := b.DropAllCollections(); err != nil {
		a.logger.Errorf("Dropping all collections failed: %v", err)
		return err
	}

	return nil
}

func (a *App) DropAllBackends() Error {
	a.logger.Infof("Dropping all backends")
	for name := range a.backends {
		if err := a.DropBackend(name); err != nil {
			return err
		}
	}
	a.logger.Infof("Successfully dropped all collections")
	return nil
}

func (a *App) RebuildBackend(name string) Error {
	b := a.Backend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	a.logger.Infof("Rebuilding backend " + name)

	if err := a.DropBackend(name); err != nil {
		return err
	}

	if err := a.MigrateBackend(name, 0, false); err != nil {
		a.logger.Errorf("Migration failed: %v", err)
		return AppError{
			Code:    "backend_migration_failed",
			Message: err.Error(),
		}
	}

	return nil
}

func (a *App) RebuildAllBackends() Error {
	a.logger.Infof("Rebuilding all backends")
	for key := range a.backends {
		if err := a.RebuildBackend(key); err != nil {
			return err
		}
	}

	a.logger.Infof("Successfully migrated all backends")

	return nil
}

func (a App) UserHandler() kit.UserService {
	return a.userService
}
