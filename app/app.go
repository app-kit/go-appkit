package app

import (
	"log"
	"os"
	"io/ioutil"
	"net/http"

	"github.com/manyminds/api2go"
	"github.com/olebedev/config"

	kit "github.com/theduke/appkit"
	"github.com/theduke/appkit/servers"
)

type App struct {
	Debug bool
	ENV string
	Config *config.Config

	DefaultBackend kit.Backend
	backends map[string]kit.Backend

	resources map[string]kit.ApiResource
	userHandler kit.ApiUserHandler
}

func NewApp(cfgPath string) *App {
	app := App{}
	app.resources = make(map[string]kit.ApiResource)
	app.backends = make(map[string]kit.Backend)

	app.ReadConfig(cfgPath)

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

	if envConf, _ := cfg.Get(env); envConf != nil {
		cfg = envConf
	}

	a.Config = cfg
}

func (a *App) Run() {
	api := api2go.NewAPI("api")

	for key := range a.resources {
		res := a.resources[key]
		api.AddResource(res.GetModel(), servers.Api2GoResource{
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

func (a *App) RegisterBackend(name string, b kit.Backend) {
	a.backends[name] = b
	if a.DefaultBackend == nil {
		a.DefaultBackend = b
	}
}

func (a *App) GetBackend(name string) kit.Backend {
	b, ok := a.backends[name]
	if !ok {
		panic("Unknown backend: " + name)
	}

	return b
}

func (a *App) RegisterResource(res kit.ApiResource) {
	if res.GetBackend() == nil {
		if a.DefaultBackend == nil {
			panic("Registering resource without backend, but no default backend set.")
		}

		log.Printf("Using default backend %v for resource %v", 
			a.DefaultBackend.GetName(), res.GetModel().GetName())

		// Register model with the backend.
		a.DefaultBackend.RegisterModel(res.GetModel())

		// Set backend.
		res.SetBackend(a.DefaultBackend)
	}
	a.resources[res.GetModel().GetName()] = res

	if res.GetUserHandler() == nil {
		res.SetUserHandler(a.userHandler)
	}
}

func (a App) GetResource(name string) kit.ApiResource {
	r, ok := a.resources[name]
	if !ok {
		panic("Unknown resource: " + name)
	}

	return r
}

func (a *App) RegisterUserHandler(h kit.ApiUserHandler) {
	a.userHandler = h
	a.RegisterResource(h.GetUserResource())
	a.RegisterResource(h.GetSessionResource())

	auth := h.GetAuthItemResource()
	if auth.GetBackend() == nil {
		a.DefaultBackend.RegisterModel(auth.GetModel())
		auth.SetBackend(a.DefaultBackend)
	}
}

func (a App) GetUserHandler() kit.ApiUserHandler {
	return a.userHandler
}
