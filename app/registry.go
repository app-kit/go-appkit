package app

import (
	"github.com/Sirupsen/logrus"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
)

type Registry struct {
	app      kit.App
	logger   *logrus.Logger
	eventBus kit.EventBus
	config   kit.Config

	defaultCache kit.Cache
	caches       map[string]kit.Cache

	defaultBackend db.Backend
	backends       map[string]db.Backend

	resources map[string]kit.Resource

	frontends map[string]kit.Frontend

	methods map[string]kit.Method

	defaultSerializer kit.Serializer
	serializers       map[string]kit.Serializer

	emailService    kit.EmailService
	fileService     kit.FileService
	resourceService kit.ResourceService
	userService     kit.UserService
	templateEngine  kit.TemplateEngine
	taskService     kit.TaskService

	values map[string]interface{}
}

// Ensure Registry implements kit.Registry.
var _ kit.Registry = (*Registry)(nil)

func NewRegistry() kit.Registry {
	return &Registry{
		caches:      make(map[string]kit.Cache),
		backends:    make(map[string]db.Backend),
		resources:   make(map[string]kit.Resource),
		frontends:   make(map[string]kit.Frontend),
		methods:     make(map[string]kit.Method),
		serializers: make(map[string]kit.Serializer),
		values:      make(map[string]interface{}),
	}
}

/**
 * App.
 */

func (d *Registry) App() kit.App {
	return d.app
}

func (d *Registry) SetApp(x kit.App) {
	d.app = x
}

/**
 * Logger.
 */

func (d *Registry) Logger() *logrus.Logger {
	return d.logger
}

func (d *Registry) SetLogger(l *logrus.Logger) {
	d.logger = l
}

/**
 * EventBus.
 */

func (r *Registry) EventBus() kit.EventBus {
	return r.eventBus
}

func (r *Registry) SetEventBus(x kit.EventBus) {
	r.eventBus = x
}

/**
 * Config.
 */

func (d *Registry) Config() kit.Config {
	return d.config
}

func (d *Registry) SetConfig(c kit.Config) {
	d.config = c
}

/**
 * Caches.
 */

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

/**
 * Backends.
 */

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

func (d *Registry) AllModelInfo() map[string]*db.ModelInfo {
	info := make(map[string]*db.ModelInfo)

	for _, backend := range d.backends {
		for name, mInfo := range backend.ModelInfos() {
			info[name] = mInfo
		}
	}

	return info
}

/**
 * Resources.
 */

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

/**
 * Frontends.
 */

func (d *Registry) Frontend(name string) kit.Frontend {
	return d.frontends[name]
}

func (d *Registry) Frontends() map[string]kit.Frontend {
	return d.frontends
}

func (d *Registry) AddFrontend(frontend kit.Frontend) {
	d.frontends[frontend.Name()] = frontend
}

func (d *Registry) SetFrontends(frontends map[string]kit.Frontend) {
	d.frontends = frontends
}

func (d *Registry) HttpFrontend() kit.HttpFrontend {
	frontend := d.Frontend("http")
	if frontend == nil {
		return nil
	}
	return frontend.(kit.HttpFrontend)
}

/**
 * Methods.
 */

func (d *Registry) Method(name string) kit.Method {
	return d.methods[name]
}

func (d *Registry) Methods() map[string]kit.Method {
	return d.methods
}

func (d *Registry) AddMethod(method kit.Method) {
	d.methods[method.GetName()] = method
}

func (d *Registry) SetMethods(methods map[string]kit.Method) {
	d.methods = methods
}

/**
 * Serializers.
 */

func (d *Registry) DefaultSerializer() kit.Serializer {
	return d.defaultSerializer
}

func (d *Registry) SetDefaultSerializer(s kit.Serializer) {
	d.defaultSerializer = s
}

func (d *Registry) Serializer(name string) kit.Serializer {
	return d.serializers[name]
}

func (d *Registry) Serializers() map[string]kit.Serializer {
	return d.serializers
}

func (d *Registry) AddSerializer(serializer kit.Serializer) {
	d.serializers[serializer.Name()] = serializer

	if d.defaultSerializer == nil {
		d.defaultSerializer = serializer
	}
}

func (d *Registry) SetSerializers(serializers map[string]kit.Serializer) {
	d.serializers = serializers
}

/**
 * Services.
 */

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

func (d *Registry) TaskService() kit.TaskService {
	return d.taskService
}

func (d *Registry) SetTaskService(s kit.TaskService) {
	d.taskService = s
}

/**
 * Custom registrations.
 */

func (d *Registry) Get(name string) interface{} {
	return d.values[name]
}

func (d *Registry) Set(name string, val interface{}) {
	d.values[name] = val
}
