package app

import (
	"github.com/Sirupsen/logrus"
	"github.com/olebedev/config"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Dependencies struct {
	logger          *logrus.Logger
	config          *config.Config
	caches          map[string]kit.Cache
	defaultBackend  db.Backend
	backends        map[string]db.Backend
	resources       map[string]kit.Resource
	emailService    kit.EmailService
	fileService     kit.FileService
	resourceService kit.ResourceService
	userService     kit.UserService
	templateEngine  kit.TemplateEngine
}

// Ensure Dependencies implements kit.Dependencies.
var _ kit.Dependencies = (*Dependencies)(nil)

func NewDependencies() kit.Dependencies {
	return &Dependencies{
		caches:    make(map[string]kit.Cache),
		backends:  make(map[string]db.Backend),
		resources: make(map[string]kit.Resource),
	}
}

func (d *Dependencies) Logger() *logrus.Logger {
	return d.logger
}

func (d *Dependencies) SetLogger(l *logrus.Logger) {
	d.logger = l
}

func (d *Dependencies) Config() *config.Config {
	return d.config
}

func (d *Dependencies) SetConfig(c *config.Config) {
	d.config = c
}

func (d *Dependencies) Cache(name string) kit.Cache {
	return d.caches[name]
}

func (d *Dependencies) Caches() map[string]kit.Cache {
	return d.caches
}

func (d *Dependencies) AddCache(cache kit.Cache) {
	d.caches[cache.Name()] = cache
}

func (d *Dependencies) SetCaches(caches map[string]kit.Cache) {
	d.caches = caches
}

func (d *Dependencies) DefaultBackend() db.Backend {
	return d.defaultBackend
}

func (d *Dependencies) SetDefaultBackend(b db.Backend) {
	d.defaultBackend = b
}

func (d *Dependencies) Backend(name string) db.Backend {
	return d.backends[name]
}

func (d *Dependencies) Backends() map[string]db.Backend {
	return d.backends
}

func (d *Dependencies) AddBackend(b db.Backend) {
	d.backends[b.GetName()] = b
	if d.defaultBackend == nil {
		d.defaultBackend = b
	}
}

func (d *Dependencies) SetBackends(backends map[string]db.Backend) {
	d.backends = backends
}

func (d *Dependencies) Resource(name string) kit.Resource {
	return d.resources[name]
}

func (d *Dependencies) Resources() map[string]kit.Resource {
	return d.resources
}

func (d *Dependencies) AddResource(res kit.Resource) {
	d.resources[res.Collection()] = res
}

func (d *Dependencies) SetResources(resources map[string]kit.Resource) {
	d.resources = resources
}

func (d *Dependencies) EmailService() kit.EmailService {
	return d.emailService
}

func (d *Dependencies) SetEmailService(s kit.EmailService) {
	d.emailService = s
}

func (d *Dependencies) FileService() kit.FileService {
	return d.fileService
}

func (d *Dependencies) SetFileService(s kit.FileService) {
	d.fileService = s
}

func (d *Dependencies) ResourceService() kit.ResourceService {
	return d.resourceService
}

func (d *Dependencies) SetResourceService(s kit.ResourceService) {
	d.resourceService = s
}

func (d *Dependencies) UserService() kit.UserService {
	return d.userService
}

func (d *Dependencies) SetUserService(s kit.UserService) {
	d.userService = s
}

func (d *Dependencies) TemplateEngine() kit.TemplateEngine {
	return d.templateEngine
}

func (d *Dependencies) SetTemplateEngine(e kit.TemplateEngine) {
	d.templateEngine = e
}
