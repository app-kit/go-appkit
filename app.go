package appkit

import (
	"log"
	"os"
	"io/ioutil"
	"net/http"
	"fmt"

	"github.com/manyminds/api2go"
	"github.com/olebedev/config"
	"github.com/spf13/cobra"
	"github.com/julienschmidt/httprouter"	

	db "github.com/theduke/go-dukedb"
)

type App struct {
	Debug bool

	ENV string
	AutoMigrate bool

	Config *config.Config

	DefaultBackend db.Backend
	backends map[string]db.Backend

	resources map[string]ApiResource
	userHandler ApiUserHandler
	fileHandler ApiFileHandler

	methods map[string]*Method

	api2go *api2go.API
	router *httprouter.Router

	Cli *cobra.Command
}

func NewApp(cfgPath string) *App {
	app := App{}
	app.resources = make(map[string]ApiResource)
	app.backends = make(map[string]db.Backend)
	app.methods = make(map[string]*Method)

	app.api2go = api2go.NewAPI("api")
	app.router = app.api2go.Router()

	app.InitCli()
	app.ReadConfig(cfgPath)

	return &app
}

func (a *App) Router() *httprouter.Router {
	return a.router
}

func (a *App) ReadConfig(path string) {
	if path == "" {
		path  = "conf.yaml"
	}

	var cfg *config.Config	

	if f, err := os.Open(path); err != nil {
		log.Printf("Could not find or read config at '%v' - Using default settings\n", path)
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
		log.Printf("No environment specified, defaulting to 'dev'\n")
		a.Debug = true
	}
	a.ENV = env


	if envConf, _ := cfg.Get(env); envConf != nil {
		cfg = envConf
	}

	// Fill in default values into the config and ensure they are valid.

	// Ensure a tmp directory exists and is readable.
	tmpDir := cfg.UString("tmpDir", "tmp")
	if err := os.MkdirAll(tmpDir, 0777); err != nil {
		panic(fmt.Sprintf("Could not read or create tmp dir at '%v': %v", tmpDir, err))
	}

	a.Config = cfg
}

func (a *App) PrepareBackends() {
	for name := range a.backends {
		a.backends[name].BuildRelationshipInfo()
	}
}

func (a *App) PrepareForRun() {
	a.PrepareBackends()

	// Auto migrate if enabled or not explicitly disabled and env is debug.
	if auto, err := a.Config.Bool("autoMigrate"); (err == nil && auto) || (err != nil && a.ENV=="dev") {
		if err := a.MigrateAllBackends(false); err != nil {
			log.Printf("Migration FAILED: %v\n", err)
		}
	}
}

func (a *App) Run() {
	a.PrepareForRun()

	// Register all method routes.
	for key := range a.methods {
		method := a.methods[key]

		// Use both POST and GET to allow for easier debugging.
		a.router.GET("/api/method/" + method.Name, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			JsonWrapHandler(w, r, a, method)
		})
		a.router.POST("/api/method/" + method.Name, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			JsonWrapHandler(w, r, a, method)
		})
	}

	// Register api2json resources.
	for key := range a.resources {
		res := a.resources[key]
		a.api2go.AddResource(res.GetModel(), Api2GoResource{
			AppResource: res,
			App: a,
		})
	}

	url := a.Config.UString("host", "localhost") + ":" + a.Config.UString("port", "8000")
	log.Printf("Serving on %v", url)
	
	handler := a.api2go.Handler()
	err := http.ListenAndServe(url, handler)
	if err != nil {
		log.Printf("Could not start server: %v\n", err)
	}
}

func (a *App) RegisterBackend(name string, b db.Backend) {
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

func (a *App) RegisterMethod(method *Method) {
	if _, exists := a.methods[method.Name]; exists {
		panic(fmt.Sprintf("Method name '%v' already registered.", method.Name))
	}	

	a.methods[method.Name] = method
}

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
			resRoutes.HttpRoutes(res, a.router)
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

func (a *App) MigrateBackend(name string, version int, force bool) ApiError {
	log.Printf("MIGRATE: Migrating backend '%v'", name)
	backend := a.GetBackend(name)
	if backend == nil {
		return Error{
			Code: "unknown_backend",
			Message: fmt.Sprint("The backend '%v' does not exist", name),
		}
	}

	migrationBackend, ok := backend.(db.MigrationBackend)
	if !ok {
		return Error{
			Code: "backend_cant_migrate",
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

func (a *App) MigrateAllBackends(force bool) ApiError {
	log.Printf("MIGRATE: Migrating all backends to newest version")
	for key := range a.backends {
		if err := a.MigrateBackend(key, 0, force); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) DropBackend(name string) ApiError {
	b := a.GetBackend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	log.Printf("Dropping all collections on backend " + name)

	if err := b.DropAllCollections(); err != nil {
		log.Printf("Dropping all collections failed: %v", err)
		return err
	}

	return nil
}

func (a *App) DropAllBackends() ApiError {
	log.Printf("Dropping all backends")
	for name := range a.backends {
		if err := a.DropBackend(name); err != nil {
			return err
		}
	}
	log.Printf("Successfully dropped all collections")
	return nil
}

func (a *App) RebuildBackend(name string) ApiError {
	b := a.GetBackend(name)
	if b == nil {
		panic("Unknown backend " + name)
	}

	log.Printf("Rebuilding backend " + name)

	if err := a.DropBackend(name); err != nil {
		return err
	}
	
	if err := a.MigrateBackend(name, 0, false); err != nil {
		log.Printf("Migration failed: %v", err)
		return Error{
			Code: "backend_migration_failed",
			Message: err.Error(),
		}
	}

	return nil
}

func (a *App) RebuildAllBackends() ApiError {
	log.Printf("Rebuilding all backends")
	for key := range a.backends {
		if err := a.RebuildBackend(key); err != nil {
			return err
		}
	}

	log.Printf("Successfully migrated all backends")

	return nil
}

func (a App) GetUserHandler() ApiUserHandler {
	return a.userHandler
}
