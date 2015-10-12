package app

import (
	"github.com/Sirupsen/logrus"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Registry struct {
	logger          *logrus.Logger
	config          kit.Config
	defaultCache    kit.Cache
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

// Ensure Registry implements kit.Registry.
var _ kit.Registry = (*Registry)(nil)

func NewRegistry() kit.Registry {
	return &Registry{
		caches:    make(map[string]kit.Cache),
		backends:  make(map[string]db.Backend),
		resources: make(map[string]kit.Resource),
	}
}

func (d *Registry) Logger() *logrus.Logger {
	return d.logger
}

func (d *Registry) SetLogger(l *logrus.Logger) {
	d.logger = l
}

func (d *Registry) Config() kit.Config {
	return d.config
}

func (d *Registry) SetConfig(c kit.Config) {
	d.config = c
}

func (d *Registry) Cache(name string) kit.Cache {
	return d.caches[name]
}

func (d *Registry) DefaultCache() kit.Cache {
	return d.defaultCache
}

func (d *Registry) SetDefaultCache(c kit.Cache) {
	d.defaultCache = c
}

func (d *Registry) Caches() map[string]kit.Cache {
	return d.caches
}

func (d *Registry) AddCache(cache kit.Cache) {
	d.caches[cache.Name()] = cache
	if d.defaultCache == nil {
		d.defaultCache = cache
	}
}

func (d *Registry) SetCaches(caches map[string]kit.Cache) {
	d.caches = caches
}

func (d *Registry) DefaultBackend() db.Backend {
	return d.defaultBackend
}

func (d *Registry) SetDefaultBackend(b db.Backend) {
	d.defaultBackend = b
}

func (d *Registry) Backend(name string) db.Backend {
	return d.backends[name]
}

func (d *Registry) Backends() map[string]db.Backend {
	return d.backends
}

func (d *Registry) AddBackend(b db.Backend) {
	d.backends[b.Name()] = b
	if d.defaultBackend == nil {
		d.defaultBackend = b
	}
}

func (d *Registry) SetBackends(backends map[string]db.Backend) {
	d.backends = backends
}

func (d *Registry) Resource(name string) kit.Resource {
	return d.resources[name]
}

func (d *Registry) Resources() map[string]kit.Resource {
	return d.resources
}

func (d *Registry) AddResource(res kit.Resource) {
	d.resources[res.Collection()] = res
}

func (d *Registry) SetResources(resources map[string]kit.Resource) {
	d.resources = resources
}

func (d *Registry) EmailService() kit.EmailService {
	return d.emailService
}

func (d *Registry) SetEmailService(s kit.EmailService) {
	d.emailService = s
}

func (d *Registry) FileService() kit.FileService {
	return d.fileService
}

func (d *Registry) SetFileService(s kit.FileService) {
	d.fileService = s
}

func (d *Registry) ResourceService() kit.ResourceService {
	return d.resourceService
}

func (d *Registry) SetResourceService(s kit.ResourceService) {
	d.resourceService = s
}

func (d *Registry) UserService() kit.UserService {
	return d.userService
}

func (d *Registry) SetUserService(s kit.UserService) {
	d.userService = s
}

func (d *Registry) TemplateEngine() kit.TemplateEngine {
	return d.templateEngine
}

func (d *Registry) SetTemplateEngine(e kit.TemplateEngine) {
	d.templateEngine = e
}
