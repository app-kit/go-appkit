package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Sirupsen/logrus"

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

	apphttp "github.com/theduke/go-appkit/frontends/http"
	"github.com/theduke/go-appkit/frontends/jsonapi"
	"github.com/theduke/go-appkit/frontends/rest"
	"github.com/theduke/go-appkit/frontends/wamp"

	jsonapiserializer "github.com/theduke/go-appkit/serializers/jsonapi"

	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/email/gomail"
	emaillog "github.com/theduke/go-appkit/email/log"
)

type App struct {
	// Unique app instance ID.
	instanceID string

	registry kit.Registry

	sessionManager *SessionManager

	Cli *cobra.Command

	shutDownChannel chan bool

	// defaults is a flag indicating whether default services, frontends, etc should be built.
	defaults bool
}

// Ensure App implements App interface.
var _ kit.App = (*App)(nil)

func NewPlainApp() *App {
	app := &App{
		registry:        NewRegistry(),
		shutDownChannel: make(chan bool),
	}
	app.registry.SetApp(app)
	app.BuildDefaultLogger()

	return app
}

func NewApp(cfgPaths ...string) *App {
	app := NewPlainApp()

	configPath := "config.yaml"
	if len(cfgPaths) > 0 {
		configPath = cfgPaths[0]
	}
	app.ReadConfig(configPath)

	app.InitCli()

	app.defaults = true
	app.Defaults()

	return app
}

func (a *App) InstanceID() string {
	return a.instanceID
}

func (a *App) SetInstanceID(x string) {
	a.instanceID = x
}

func (a *App) Defaults() {
	// EmailService setup.
	a.buildEmailService()

	// Register fs cache.
	a.BuildDefaultCache()

	a.BuildDefaultFrontends()
	a.BuildDefaultMethods()
}

func (a *App) BuildDefaultMethods() {
	a.RegisterMethod(createMethod())
	a.RegisterMethod(updateMethod())
	a.RegisterMethod(deleteMethod())
	a.RegisterMethod(queryMethod())
}

func (a *App) BuildDefaultFrontends() {
	// Frontends.
	a.RegisterFrontend(apphttp.New(a.registry))
	a.RegisterFrontend(jsonapi.New(a.registry))
	a.RegisterFrontend(rest.New(a.registry))
	a.RegisterFrontend(wamp.New(a.registry))
}

func (a *App) BuildDefaultSerializers() {
	a.RegisterSerializer(jsonapiserializer.New(a.registry.Backends()))
}

func (a *App) BuildDefaultLogger() {
	// Configure logger.
	a.SetLogger(&logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	})
}

func (a *App) BuildDefaultTaskService(b db.Backend) {
	if !a.Config().UBool("tasks.enabled", false) {
		return
	}

	s := tasks.NewService(a.registry, b)
	max := a.Config().UInt("tasks.maximumConcurrentTasks", 10)
	s.SetMaximumConcurrentTasks(max)

	a.registry.SetTaskService(s)
}

func (a *App) BuildDefaultCache() {
	// Build cache.
	dir := a.registry.Config().UString("caches.fs.dir")
	if dir == "" {
		dir = a.registry.Config().TmpDir() + "/" + "cache"
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
	for name, frontend := range a.registry.Frontends() {
		if err := frontend.Init(); err != nil {
			a.Logger().Panicf("Error while initializing frontend %v: %v", name, err)
		}
	}

	if a.defaults {
		a.BuildDefaultSerializers()
	}
}

func (a *App) Run() {
	a.PrepareForRun()

	// Run taskrunner if possible.
	if service := a.Registry().TaskService(); service != nil {
		if runner, ok := service.(kit.TaskRunner); ok {
			if err := runner.Run(); err != nil {
				panic("Could not start task runner: " + err.Error())
			}
		}
	}

	// Run the session manager.
	a.sessionManager = NewSessionManager(a)
	a.sessionManager.Run()

	// Run frontends.

	for name, frontend := range a.registry.Frontends() {
		if err := frontend.Start(); err != nil {
			a.Logger().Panicf("Could not start frontend %v: %v", name, err)
		}
	}

	// Crawl on startup if enabled.
	if a.Config().UBool("crawler.onRun", false) {
		a.RunCrawler()
	}

	<-a.shutDownChannel
}

func (a *App) Shutdown() (shutdownChan chan bool, err apperror.Error) {
	return nil, nil
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
 * Backends.
 */

func (a *App) RegisterBackend(b db.Backend) {
	if b.GetLogger() == nil {
		b.SetLogger(a.Logger())
	}

	isDefault := a.DefaultBackend() == nil
	a.registry.AddBackend(b)

	// If no backend was registered before, create a default UserService and FileService.
	if isDefault && a.defaults {
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
	a.registry.AddMethod(method)
}

func (a *App) RunMethod(name string, r kit.Request, responder func(kit.Response), withFinishedChannel bool) (chan bool, apperror.Error) {
	method := a.registry.Method(name)
	if method == nil {
		return nil, &apperror.Err{
			Code:    "unknown_method",
			Message: fmt.Sprintf("The method %v does not exist", name),
		}
	}

	if r.GetSession() == nil {
		session, err := a.UserService().StartSession(r.GetUser())
		if err != nil {
			return nil, err
		}
		r.SetSession(session)
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
 * Http routes.
 */

func (a *App) RegisterHttpHandler(method, path string, handler kit.RequestHandler) {
	httpFrontend := a.registry.HttpFrontend()
	if httpFrontend == nil {
		a.Logger().Panicf("No HTTP frontend found.")
	}

	httpFrontend.RegisterHttpHandler(method, path, handler)
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
				a.RegisterHttpHandler(route.Method(), route.Route(), route.Handler())
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

func (a *App) RegisterFrontend(f kit.Frontend) {
	a.registry.AddFrontend(f)
}

/**
 * Serializers.
 */

func (a *App) RegisterSerializer(s kit.Serializer) {
	a.registry.AddSerializer(s)
}
