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
	"github.com/theduke/go-appkit/caches/fs"
	"github.com/theduke/go-appkit/crawler"
	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/tasks"
	"github.com/theduke/go-appkit/users"

	"github.com/theduke/go-appkit/frontends/jsonapi"

	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/email/gomail"
	emaillog "github.com/theduke/go-appkit/email/log"
)

type App struct {
	registry kit.Registry

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

func NewApp(cfgPaths ...string) *App {
	app := App{
		registry:          NewRegistry(),
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

	configPath := "config.yaml"
	if len(cfgPaths) > 0 {
		configPath = cfgPaths[0]
	}
	app.ReadConfig(configPath)

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

	// Register fs cache.
	a.BuildDefaultCache()

	a.RegisterFrontend(jsonapi.New(a))
}

func (a *App) BuildDefaultTaskService(b db.Backend) {
	if !a.Config().UBool("tasks.enabled", false) {
		return
	}

	s := tasks.NewService(a.registry, b)
	max := a.Config().UInt("tasks.maximumConcurrentTasks", 10)
	s.SetMaximumConcurrentTasks(max)

	if err := s.Run(); err != nil {
		panic("Could not start task runner: " + err.Error())
	}

	a.registry.SetTaskService(s)
}

func (a *App) BuildDefaultCache() {
	// Build cache.
	dir := a.Config().UString("caches.fs.dir")
	if dir == "" {
		dir = a.Config().TmpDir() + "/" + "cache"
	}
	fsCache, err := fs.New(dir)
	if err != nil {
		panic("Could not initialize fs cache: " + err.Error())
	}
	a.RegisterCache(fsCache)
}

func (a *App) BuildDefaultUserService(b db.Backend) {
	s := users.NewService(nil, b, nil)
	a.RegisterUserService(s)
}

func (a *App) BuildDefaultFileService(b db.Backend) {
	// Register file service with fs backend.

	dir := a.Config().UString("files.dir")
	if dir == "" {
		dir = a.Config().UString("dataDir", "data") + "/" + "files"
	}
	fileService := files.NewFileServiceWithFs(nil, dir)
	a.RegisterFileService(fileService)
}

func (a *App) ENV() string {
	return a.Config().ENV()
}

func (a *App) SetENV(x string) {
	a.Config().Set("env", x)
}

func (a *App) Debug() bool {
	return a.Config().Debug()
}

func (a *App) SetDebug(x bool) {
	a.Config().Set("debug", x)
}

func (a *App) Registry() kit.Registry {
	return a.registry
}

func (a *App) Logger() *logrus.Logger {
	return a.registry.Logger()
}

func (a *App) SetLogger(x *logrus.Logger) {
	a.registry.SetLogger(x)
}

func (a *App) Router() *httprouter.Router {
	return a.router
}

/**
 * Config.
 */

func (a *App) Config() kit.Config {
	return a.registry.Config()
}

func (a *App) SetConfig(x kit.Config) {
	a.registry.SetConfig(x)
}

func (a *App) ReadConfig(path string) {
	if path == "" {
		path = "config.yaml"
	}

	var cfg kit.Config

	if f, err := os.Open(path); err != nil {
		a.Logger().Infof("Could not find or read config at '%v' - Using default settings\n", path)
	} else {
		defer f.Close()
		content, err := ioutil.ReadAll(f)
		if err != nil {
			a.Logger().Panicf("Could not read config at '%v': %v\n", path, err)
		} else {
			rawCfg, err := config.ParseYaml(string(content))
			if err != nil {
				panic("Malformed config file " + path + ": " + err.Error())
			}

			cfg = NewConfig(rawCfg.Root)
		}
	}

	if cfg == nil {
		cfg = NewConfig(map[string]interface{}{
			"env":      "dev",
			"debug":    true,
			"tmpPath":  "tmp",
			"dataPath": "data",
		})
	}

	// Read environment variables.
	cfg.ENV()

	// Set default values if not present.
	env, _ := cfg.String("ENV")
	if env == "" {
		a.Logger().Info("No environment specified, defaulting to 'dev'")
		cfg.Set("ENV", "dev")
		env = "dev"
	}

	if envCfg, err := cfg.Get(env); err == nil {
		cfg = NewConfig(envCfg.GetData())
		cfg.Set("ENV", env)
	}

	// Fill in default values into the config and ensure they are valid.

	// If debug is not explicitly set, set it to false, or to true if
	// environment is dev.
	_, err := cfg.Bool("debug")
	if err != nil {
		cfg.Set("debug", env == "dev")
	}

	// Ensure a tmp directory exists and is readable.
	tmpDir := cfg.TmpDir()
	if err := os.MkdirAll(tmpDir, 0777); err != nil {
		a.Logger().Panicf("Could not read or create tmp dir at '%v': %v", tmpDir, err)
	}

	// Ensure a data directory exists and is readable.
	dataDir := cfg.UPath("dataDir", "data")
	if err := os.MkdirAll(dataDir, 0777); err != nil {
		a.Logger().Panicf("Could not read or create data dir at '%v': %v", tmpDir, err)
	}

	a.registry.SetConfig(cfg)
}

func (a *App) PrepareBackends() {
	backends := a.registry.Backends()
	for name := range backends {
		backends[name].BuildRelationshipInfo()
	}
}

func (a *App) PrepareForRun() {
	a.PrepareBackends()

	// Auto migrate if enabled or not explicitly disabled and env is debug.
	if auto, err := a.Config().Bool("autoRunMigrations"); (err == nil && auto) || (err != nil && a.ENV() == "dev") {
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

	// Handle options requests.
	a.router.OPTIONS("/api/method/:name", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		httpHandler(w, r, params, a, func(app kit.App, r kit.Request) (kit.Response, bool) {
			return &kit.AppResponse{}, false
		})
	})
	// Handle the method request.
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
	crawler.Logger = a.Logger()

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

	isDefault := a.DefaultBackend() == nil
	a.registry.AddBackend(b)

	// If no backend was registered before, create a default UserService and FileService.
	if isDefault {
		if a.UserService() == nil {
			a.BuildDefaultUserService(b)
		}
		if a.FileService() == nil {
			a.BuildDefaultFileService(b)
		}
		if a.registry.TaskService() == nil {
			a.BuildDefaultTaskService(b)
		}
	}
}

func (a *App) Backend(name string) db.Backend {
	return a.registry.Backend(name)
}

func (a *App) DefaultBackend() db.Backend {
	return a.registry.DefaultBackend()
}

/**
 * Caches.
 */

func (a *App) RegisterCache(c kit.Cache) {
	a.registry.AddCache(c)
}

func (a *App) Cache(name string) kit.Cache {
	return a.registry.Cache(name)
}

/**
 * Email service.
 */

func (a *App) buildEmailService() {
	host := a.registry.Config().UString("email.host")
	port := a.registry.Config().UInt("email.port")
	user := a.registry.Config().UString("email.user")
	pw := a.registry.Config().UString("email.password")

	fromEmail := a.registry.Config().UString("email.from", "no-reply@appkit")
	fromName := a.registry.Config().UString("email.fromName", "Appkit")

	from := email.Recipient{
		Email: fromEmail,
		Name:  fromName,
	}

	if host != "" && port > 0 && user != "" && pw != "" {
		a.RegisterEmailService(gomail.New(a.registry, host, port, user, pw, fromEmail, fromName))
		a.Logger().Debug("Using gomail email service")
	} else {
		a.RegisterEmailService(emaillog.New(a.registry, from))
		a.Logger().Debug("Using log email service")
	}
}

func (a *App) RegisterEmailService(s kit.EmailService) {
	if s.Registry() == nil {
		s.SetRegistry(a.registry)
	}
	s.SetDebug(a.Debug())
	a.registry.SetEmailService(s)
}

func (a *App) EmailService() kit.EmailService {
	return a.registry.EmailService()
}

/**
 * TemplateEngine.
 */

func (a *App) RegisterTemplateEngine(e kit.TemplateEngine) {
	a.registry.SetTemplateEngine(e)
}

func (a *App) TemplateEngine() kit.TemplateEngine {
	return a.registry.TemplateEngine()
}

/**
 * Methods.
 */

func (a *App) RegisterMethod(method kit.Method) {
	if _, exists := a.methods[method.GetName()]; exists {
		a.Logger().Warnf("Overwriting already registered method '%v'.", method.GetName())
	}

	a.methods[method.GetName()] = method
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
		if a.registry.DefaultBackend() == nil {
			a.Logger().Panic("Registering resource without backend, but no default backend set.")
		}

		// Set backend.
		res.SetBackend(a.registry.DefaultBackend())
	}

	if res.Registry() == nil {
		res.SetRegistry(a.registry)
	}

	res.SetDebug(a.Debug())

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

	a.registry.AddResource(res)
}

func (a App) Resource(name string) kit.Resource {
	return a.registry.Resource(name)
}

/**
 * UserHandler.
 */

func (a *App) RegisterUserService(s kit.UserService) {
	if s.Registry() == nil {
		s.SetRegistry(a.registry)
	}

	s.SetDebug(a.Debug())

	a.RegisterResource(s.UserResource())
	a.RegisterResource(s.SessionResource())
	a.RegisterResource(s.RoleResource())
	a.RegisterResource(s.PermissionResource())

	a.registry.SetUserService(s)
}

func (a *App) UserService() kit.UserService {
	return a.registry.UserService()
}

/**
 * FileService.
 */

func (a *App) RegisterFileService(f kit.FileService) {
	r := f.Resource()
	if r == nil {
		a.Logger().Panic("Trying to register file handler without resource")
	}

	if f.Registry() == nil {
		f.SetRegistry(a.registry)
	}
	f.SetDebug(a.Debug())

	a.RegisterResource(r)
	a.registry.SetFileService(f)
}

func (a *App) FileService() kit.FileService {
	return a.registry.FileService()
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
	backends := a.registry.Backends()
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
	for name := range a.registry.Backends() {
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
	for key := range a.registry.Backends() {
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
