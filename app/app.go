package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/olebedev/config"
	"github.com/spf13/cobra"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	//"github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/crawler"
	"github.com/theduke/go-appkit/resources"

	"github.com/theduke/go-appkit/frontends/jsonapi"

	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/email/gomail"
	emaillog "github.com/theduke/go-appkit/email/log"
)

type App struct {
	env   string
	debug bool

	logger *logrus.Logger

	deps      kit.Dependencies
	frontends map[string]kit.Frontend

	methods map[string]kit.Method

	beforeMiddlewares []kit.RequestHandler
	afterMiddlewares  []kit.AfterRequestMiddleware

	serverErrorHandler kit.AfterRequestMiddleware
	notFoundHandler    kit.RequestHandler

	sessionManager *SessionManager

	router *httprouter.Router

	Cli *cobra.Command

	shutDownChannel chan bool
}

// Ensure App implements App interface.
var _ kit.App = (*App)(nil)

func NewApp(cfgPath string) *App {
	app := App{
		deps:              NewDependencies(),
		frontends:         make(map[string]kit.Frontend),
		methods:           make(map[string]kit.Method),
		beforeMiddlewares: make([]kit.RequestHandler, 0),
		afterMiddlewares:  make([]kit.AfterRequestMiddleware, 0),

		router: httprouter.New(),

		shutDownChannel: make(chan bool),
	}

	// Configure logger.
	app.SetLogger(&logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	})

	app.InitCli()
	app.ReadConfig(cfgPath)

	app.Defaults()

	return &app
}

func (a *App) Defaults() {
	// EmailService setup.
	a.buildEmailService()

	a.RegisterMethod(createMethod())
	a.RegisterMethod(updateMethod())
	a.RegisterMethod(deleteMethod())
	a.RegisterMethod(queryMethod())

	a.RegisterBeforeMiddleware(RequestTraceMiddleware)
	a.RegisterBeforeMiddleware(AuthenticationMiddleware)

	a.RegisterAfterMiddleware(ServerErrorMiddleware)
	a.RegisterAfterMiddleware(RequestTraceAfterMiddleware)
	a.RegisterAfterMiddleware(RequestLoggerMiddleware)

	a.RegisterFrontend(jsonapi.New(a))
}

func (a *App) ENV() string {
	return a.env
}

func (a *App) SetENV(x string) {
	a.env = x
}

func (a *App) Debug() bool {
	return a.debug
}

func (a *App) SetDebug(x bool) {
	a.debug = x
}

func (a *App) Dependencies() kit.Dependencies {
	return a.deps
}

func (a *App) Logger() *logrus.Logger {
	return a.logger
}

func (a *App) SetLogger(x *logrus.Logger) {
	a.logger = x
	a.deps.SetLogger(x)
}

func (a *App) Router() *httprouter.Router {
	return a.router
}

func (a *App) TmpDir() string {
	return a.deps.Config().UString("tmpDir", "tmp")
}

/**
 * Config.
 */

func (a *App) Config() *config.Config {
	return a.deps.Config()
}

func (a *App) SetConfig(x *config.Config) {
	a.deps.SetConfig(x)
}

func (a *App) ReadConfig(path string) {
	if path == "" {
		path = "config.yaml"
	}

	var cfg *config.Config

	if f, err := os.Open(path); err != nil {
		a.Logger().Infof("Could not find or read config at '%v' - Using default settings\n", path)
	} else {
		defer f.Close()
		content, err := ioutil.ReadAll(f)
		if err != nil {
			a.Logger().Panicf("Could not read config at '%v': %v\n", path, err)
		} else {
			cfg, err = config.ParseYaml(string(content))
			if err != nil {
				a.Logger().Panicf("YAML error while parsing config at '%v': %v\n", path, err)
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
		a.Logger().Info("No environment specified, defaulting to 'dev'")
		a.debug = true
	}
	a.env = env

	// Fill in default values into the config and ensure they are valid.

	// Ensure a tmp directory exists and is readable.
	tmpDir := cfg.UString("tmpDir", "tmp")
	if err := os.MkdirAll(tmpDir, 0777); err != nil {
		a.Logger().Panicf("Could not read or create tmp dir at '%v': %v", tmpDir, err)
	}

	a.deps.SetConfig(cfg)
}

func (a *App) PrepareBackends() {
	backends := a.deps.Backends()
	for name := range backends {
		backends[name].BuildRelationshipInfo()
	}
}

func (a *App) PrepareForRun() {
	a.PrepareBackends()

	// Auto migrate if enabled or not explicitly disabled and env is debug.
	if auto, err := a.Config().Bool("autoRunMigrations"); (err == nil && auto) || (err != nil && a.env == "dev") {
		if err := a.MigrateAllBackends(false); err != nil {
			a.Logger().Errorf("Migration FAILED: %v\n", err)
		}
	}

	// Initialize frontends.
	for name := range a.frontends {
		if err := a.frontends[name].Init(); err != nil {
			a.Logger().Panicf("Error while initializing frontend %v: %v", name, err)
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
		App:     a,
		Handler: a.notFoundHandler,
	}

	// Install handler for index.
	indexTpl, err := getIndexTpl(a)
	if err != nil {
		a.Logger().Panic(err)
	}

	a.router.GET("/", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, func(kit.App, kit.Request) (kit.Response, bool) {
			return &kit.AppResponse{
				RawData: indexTpl,
			}, false
		})
	})

	// Serve files routes.
	serveFiles := a.Config().UMap("serveFiles")
	for route := range serveFiles {
		path, ok := serveFiles[route].(string)
		if !ok {
			a.Logger().Error("Config error: serveFiles configuration invalid: Must be map/dictionary with paths")
			continue
		}
		a.ServeFiles(route, path)
	}

	// Register route for method calls.
	methodHandler := func(a kit.App, r kit.Request) (kit.Response, bool) {
		r.ParseJsonData()

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

	// Run the session manager.
	a.sessionManager = NewSessionManager(a)
	a.sessionManager.Run()

	// Run frontends.
	for name := range a.frontends {
		if err := a.frontends[name].Start(); err != nil {
			a.Logger().Panicf("Could not start frontend %v: %v", name, err)
		}
	}

	url := a.Config().UString("host", "localhost") + ":" + a.Config().UString("port", "8000")
	a.Logger().Infof("Serving on %v", url)

	go func() {
		err2 := http.ListenAndServe(url, a.router)
		if err2 != nil {
			a.Logger().Panicf("Could not start server: %v\n", err)
		}
	}()

	// Crawl on startup if enabled.
	if a.Config().UBool("crawler.onRun", false) {
		a.RunCrawler()
	}

	<-a.shutDownChannel
}

/**
 * Crawling.
 */

func (a *App) RunCrawler() {
	recrawlInterval := a.Config().UInt("crawler.recrawlInterval", 0)

	a.Logger().Infof("Running crawler with recrawl interval %v", recrawlInterval)

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
	concurrentRequests := a.Config().UInt("crawler.concurrentRequests", 5)
	host := a.Config().UString("url")
	if host == "" {
		a.Logger().Error("Can't crawl because 'url' setting is not specified")
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
	a.Logger().Debugf("Serving files from directory '%v' at route '%v'", path, route)

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

func (a *App) RegisterBackend(b db.Backend) {
	if b.GetLogger() == nil {
		b.SetLogger(a.Logger())
	}
	a.deps.AddBackend(b)
}

func (a *App) Backend(name string) db.Backend {
	return a.deps.Backend(name)
}

/**
 * Caches.
 */

func (a *App) RegisterCache(c kit.Cache) {
	a.deps.AddCache(c)
}

func (a *App) Cache(name string) kit.Cache {
	return a.deps.Cache(name)
}

/**
 * Email service.
 */

func (a *App) buildEmailService() {
	host := a.deps.Config().UString("email.host")
	port := a.deps.Config().UInt("email.port")
	user := a.deps.Config().UString("email.user")
	pw := a.deps.Config().UString("email.password")

	fromEmail := a.deps.Config().UString("email.from", "no-reply@appkit")
	fromName := a.deps.Config().UString("email.fromName", "Appkit")

	from := email.Recipient{
		Email: fromEmail,
		Name:  fromName,
	}

	if host != "" && port > 0 && user != "" && pw != "" {
		a.RegisterEmailService(gomail.New(a.deps, host, port, user, pw, fromEmail, fromName))
		a.Logger().Debug("Using gomail email service")
	} else {
		a.RegisterEmailService(emaillog.New(a.deps, from))
		a.Logger().Debug("Using log email service")
	}
}

func (a *App) RegisterEmailService(s kit.EmailService) {
	if s.Dependencies() == nil {
		s.SetDependencies(a.deps)
	}
	s.SetDebug(a.debug)
	a.deps.SetEmailService(s)
}

func (a *App) EmailService() kit.EmailService {
	return a.deps.EmailService()
}

/**
 * TemplateEngine.
 */

func (a *App) RegisterTemplateEngine(e kit.TemplateEngine) {
	a.deps.SetTemplateEngine(e)
}

func (a *App) TemplateEngine() kit.TemplateEngine {
	return a.deps.TemplateEngine()
}

/**
 * Methods.
 */

func (a *App) RegisterMethod(method kit.Method) {
	if _, exists := a.methods[method.Name()]; exists {
		a.Logger().Panicf("Method name '%v' already registered.", method.Name())
	}

	a.methods[method.Name()] = method
}

func (a *App) RunMethod(name string, r kit.Request, responder func(kit.Response), withFinishedChannel bool) (chan bool, apperror.Error) {
	if r.GetSession() == nil {
		return nil, &apperror.Err{
			Code:    "no_session",
			Message: "Can't run a method without a session",
		}
	}

	method := a.methods[name]
	if method == nil {
		return nil, &apperror.Err{
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

func (a *App) RegisterResource(res kit.Resource) {
	if res.Backend() == nil {
		if a.deps.DefaultBackend() == nil {
			a.Logger().Panic("Registering resource without backend, but no default backend set.")
		}

		// Set backend.
		res.SetBackend(a.deps.DefaultBackend())
	}

	if res.Dependencies() == nil {
		res.SetDependencies(a.deps)
	}

	res.SetDebug(a.debug)

	// Allow a resource to register custom http routes and methods.
	if res.Hooks() != nil {
		// Handle http routes.
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

		// Handle methods.
		if resMethods, ok := res.Hooks().(resources.MethodsHook); ok {
			for _, method := range resMethods.Methods(res) {
				a.RegisterMethod(method)
			}
		}
	}

	a.deps.AddResource(res)
}

func (a App) Resource(name string) kit.Resource {
	return a.deps.Resource(name)
}

/**
 * UserHandler.
 */

func (a *App) RegisterUserService(s kit.UserService) {
	if s.Dependencies() == nil {
		s.SetDependencies(a.deps)
	}

	s.SetDebug(a.debug)

	a.RegisterResource(s.UserResource())
	a.RegisterResource(s.SessionResource())
	a.RegisterResource(s.RoleResource())
	a.RegisterResource(s.PermissionResource())

	a.deps.SetUserService(s)
}

func (a *App) UserService() kit.UserService {
	return a.deps.UserService()
}

/**
 * FileService.
 */

func (a *App) RegisterFileService(f kit.FileService) {
	r := f.Resource()
	if r == nil {
		a.Logger().Panic("Trying to register file handler without resource")
	}

	if f.Dependencies() == nil {
		f.SetDependencies(a.deps)
	}
	f.SetDebug(a.debug)

	a.RegisterResource(r)
	a.deps.SetFileService(f)
}

func (a *App) FileService() kit.FileService {
	return a.deps.FileService()
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

func (a *App) NotFoundHandler() kit.RequestHandler {
	return a.notFoundHandler
}

func (a *App) SetNotFoundHandler(x kit.RequestHandler) {
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

func (a *App) MigrateBackend(name string, version int, force bool) apperror.Error {
	a.Logger().Infof("MIGRATE: Migrating backend '%v'", name)
	backend := a.Backend(name)
	if backend == nil {
		return &apperror.Err{
			Code:    "unknown_backend",
			Message: fmt.Sprint("The backend '%v' does not exist", name),
		}
	}

	migrationBackend, ok := backend.(db.MigrationBackend)
	if !ok {
		return &apperror.Err{
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

func (a *App) MigrateAllBackends(force bool) apperror.Error {
	a.Logger().Infof("MIGRATE: Migrating all backends to newest version")
	backends := a.deps.Backends()
	for key := range backends {
		if err := a.MigrateBackend(key, 0, force); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) DropBackend(name string) apperror.Error {
	b := a.Backend(name)
	if b == nil {
		a.Logger().Panicf("Unknown backend %v", name)
	}

	a.Logger().Infof("Dropping all collections on backend " + name)

	if err := b.DropAllCollections(); err != nil {
		a.Logger().Errorf("Dropping all collections failed: %v", err)
		return err
	}

	return nil
}

func (a *App) DropAllBackends() apperror.Error {
	a.Logger().Infof("Dropping all backends")
	for name := range a.deps.Backends() {
		if err := a.DropBackend(name); err != nil {
			return err
		}
	}
	a.Logger().Infof("Successfully dropped all collections")
	return nil
}

func (a *App) RebuildBackend(name string) apperror.Error {
	b := a.Backend(name)
	if b == nil {
		a.Logger().Panicf("Unknown backend %v", name)
	}

	a.Logger().Infof("Rebuilding backend " + name)

	if err := a.DropBackend(name); err != nil {
		return err
	}

	if err := a.MigrateBackend(name, 0, false); err != nil {
		a.Logger().Errorf("Migration failed: %v", err)
		return apperror.Wrap(err, "backend_migration_failed")
	}

	return nil
}

func (a *App) RebuildAllBackends() apperror.Error {
	a.Logger().Infof("Rebuilding all backends")
	for key := range a.deps.Backends() {
		if err := a.RebuildBackend(key); err != nil {
			return err
		}
	}

	a.Logger().Infof("Successfully migrated all backends")

	return nil
}

/**
 * Frontends.
 */

func (a *App) Frontend(name string) kit.Frontend {
	return a.frontends[name]
}

func (a *App) RegisterFrontend(f kit.Frontend) {
	a.frontends[f.Name()] = f
}
