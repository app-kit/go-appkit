package appkit

import (
	"log"
	"os"
	"io/ioutil"
	"net/http"

	"github.com/manyminds/api2go"
	"github.com/olebedev/config"
	"github.com/spf13/cobra"

	db "github.com/theduke/dukedb"
)

type App struct {
	Debug bool
	ENV string
	Config *config.Config

	DefaultBackend db.Backend
	backends map[string]db.Backend

	resources map[string]ApiResource
	userHandler ApiUserHandler

	Cli *cobra.Command
}

func NewApp(cfgPath string) *App {
	app := App{}
	app.resources = make(map[string]ApiResource)
	app.backends = make(map[string]db.Backend)

	app.InitCli()
	return &app
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
			log.Printf("Could not read config at '%v': %v\n", path, err)
		} else {
			cfg, err = config.ParseYaml(string(content))
			if err != nil {
				log.Printf("YAML error while parsing config at '%v': %v\n", path, err)
			}
		}	
	}
	
	if cfg == nil {
		c, _ := config.ParseYaml("ENV: dev\n")
		cfg = c
	}

	cfg.Flag().Env()

	env := cfg.UString("ENV")
	if env == "" {
		env = "dev"
		log.Printf("No environment specified, defaulting to 'dev'\n")
	}
	a.ENV = env

	if envConf, _ := cfg.Get(env); envConf != nil {
		cfg = envConf
	}

	a.Config = cfg
}

func (a *App) Run() {
	api := api2go.NewAPI("api")

	for key := range a.resources {
		res := a.resources[key]
		api.AddResource(res.GetModel(), Api2GoResource{
			AppResource: res,
		})
	}

	handler := api.Handler()

	url := a.Config.UString("host", "localhost") + ":" + a.Config.UString("port", "8000")
	log.Printf("Serving on %v\n", url)
	
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

func (a *App) RegisterResource(model db.Model, hooks ApiHooks) {
	res := NewResource(model, hooks)
	a.RegisterCustomResource(res)
}

func (a *App) RegisterCustomResource(res ApiResource) {
	res.SetDebug(a.Debug)
	if res.GetBackend() == nil {
		if a.DefaultBackend == nil {
			panic("Registering resource without backend, but no default backend set.")
		}

		log.Printf("Using default backend %v for resource %v", 
			a.DefaultBackend.GetName(), res.GetModel().GetCollection())

		// Set backend.
		res.SetBackend(a.DefaultBackend)
	} 

	// Register model with the backend.
	res.GetBackend().RegisterModel(res.GetModel())

	// Set userhandler if neccessary.
	if res.GetUserHandler() == nil {
		res.SetUserHandler(a.userHandler)
	}

	a.resources[res.GetModel().GetCollection()] = res
}

func (a App) GetResource(name string) ApiResource {
	r, ok := a.resources[name]
	if !ok {
		panic("Unknown resource: " + name)
	}

	return r
}

func (a *App) RegisterUserHandler(h ApiUserHandler) {
	a.userHandler = h
	a.RegisterCustomResource(h.GetUserResource())
	a.RegisterCustomResource(h.GetSessionResource())

	auth := h.GetAuthItemResource()
	if auth.GetBackend() == nil {
		a.DefaultBackend.RegisterModel(auth.GetModel())
		auth.SetBackend(a.DefaultBackend)
	}
}

func (a App) GetUserHandler() ApiUserHandler {
	return a.userHandler
}
