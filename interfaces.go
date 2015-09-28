package appkit

import (
	"io"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/olebedev/config"
	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
)

type Model interface {
	Collection() string
	GetID() interface{}
	SetID(id interface{}) error
	GetStrID() string
	SetStrID(id string) error
}

/**
 * User interfaces.
 */

type Session interface {
	Model
	UserModel

	SetType(string)
	GetType() string

	SetToken(string)
	GetToken() string

	SetStartedAt(time.Time)
	GetStartedAt() time.Time

	SetValidUntil(time.Time)
	GetValidUntil() time.Time

	IsGuest() bool
}

type Role interface {
	Model

	GetName() string
	SetName(string)

	GetPermissions() []string
	SetPermissions(permissions []string)
	AddPermission(perm ...string)
	RemovePermission(perm ...string)
	ClearPermissions()
	HasPermission(perm ...string) bool
}

type Permission interface {
	Model

	GetName() string
	SetName(string)
}

type User interface {
	Model

	SetIsActive(bool)
	IsActive() bool

	SetUsername(string)
	GetUsername() string

	SetEmail(string)
	GetEmail() string

	SetIsEmailConfirmed(bool)
	IsEmailConfirmed() bool

	SetLastLogin(time.Time)
	GetLastLogin() time.Time

	SetCreatedAt(time.Time)
	GetCreatedAt() time.Time

	SetUpdatedAt(time.Time)
	GetUpdatedAt() time.Time

	SetProfile(UserProfile)
	GetProfile() UserProfile

	GetData() (interface{}, apperror.Error)
	SetData(interface{}) apperror.Error

	GetRoles() []string
	SetRoles(roles []string)
	ClearRoles()
	AddRole(role ...string)
	RemoveRole(role ...string)
	HasRole(role ...string) bool

	HasPermission(perm ...string) bool
}

type UserModel interface {
	GetUser() User
	SetUser(User)

	GetUserID() interface{}
	SetUserID(id interface{}) error
}

type UserProfile interface {
	Model
	UserModel
}

type AuthItem interface {
	Model
	UserModel
}

type UserToken interface {
	Model
	UserModel

	GetType() string
	SetType(string)

	GetToken() string
	SetToken(string)

	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)

	IsValid() bool
}

type AuthAdaptor interface {
	Name() string

	Backend() db.Backend
	SetBackend(db.Backend)

	RegisterUser(user User, data map[string]interface{}) (AuthItem, apperror.Error)

	// Authenticate  a user based on data map, and return userID or an error.
	// The userID argument may be an empty string if the adaptor has to
	// map the userID.
	Authenticate(userID string, data map[string]interface{}) (string, apperror.Error)
}

/**
 * Api interfaces.
 */

type Request interface {
	GetContext() *Context
	GetMeta() Context

	GetData() interface{}
	GetRawData() []byte

	// Parse json contained in RawData and extract data and meta.
	ParseJsonData() apperror.Error

	GetUser() User
	SetUser(User)

	GetSession() Session
	SetSession(Session)

	GetHttpRequest() *http.Request
	SetHttpRequest(request *http.Request)

	ReadHtmlBody() apperror.Error

	GetHttpResponseWriter() http.ResponseWriter
	SetHttpResponseWriter(writer http.ResponseWriter)
}

type Response interface {
	GetError() apperror.Error

	GetHttpStatus() int
	SetHttpStatus(int)

	GetMeta() map[string]interface{}
	SetMeta(map[string]interface{})

	GetData() interface{}
	SetData(interface{})

	GetRawData() []byte
	SetRawData([]byte)

	GetRawDataReader() io.ReadCloser
	SetRawDataReader(io.ReadCloser)
}

type RequestHandler func(App, Request) (Response, bool)
type AfterRequestMiddleware func(App, Request, Response) bool

type HttpRoute interface {
	Route() string
	Method() string
	Handler() RequestHandler
}

/**
 * Caches.
 */

type CacheItem interface {
	Model

	GetKey() string
	SetKey(string)

	GetValue() interface{}
	SetValue(interface{})

	ToString() (string, apperror.Error)
	FromString(string) apperror.Error

	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)
	IsExpired() bool

	GetTags() []string
	SetTags([]string)
}

type Cache interface {
	Name() string
	SetName(string)

	// Save a new item into the cache.
	Set(CacheItem) apperror.Error
	SetString(key string, value string, expiresAt *time.Time, tags []string) apperror.Error

	// Retrieve a cache item from the cache.
	Get(key string, item ...CacheItem) (CacheItem, apperror.Error)
	GetString(key string) (string, apperror.Error)

	// Delete item from the cache.
	Delete(key ...string) apperror.Error

	// Get all keys stored in the cache.
	Keys() ([]string, apperror.Error)

	// Return all keys that have a certain tag.
	KeysByTags(tag ...string) ([]string, apperror.Error)

	// Clear all items from the cache.
	Clear() apperror.Error

	// Clear all items with the specified tags.
	ClearTag(tag string) apperror.Error

	// Clean up all expired entries.
	Cleanup() apperror.Error
}

/**
 * Emails.
 */

type EmailRecipient interface {
	GetEmail() string
	GetName() string
}

type EmailPart interface {
	GetMimeType() string
	GetContent() []byte
	GetFilePath() string
	GetReader() io.ReadCloser
}

type Email interface {
	SetFrom(email, name string)
	GetFrom() EmailRecipient

	AddTo(email, name string)
	GetTo() []EmailRecipient

	AddCc(email, name string)
	GetCc() []EmailRecipient

	AddBcc(email, name string)
	GetBcc() []EmailRecipient

	SetSubject(string)
	GetSubject() string

	SetBody(contentType string, body []byte)
	AddBody(contentType string, body []byte)
	GetBodyParts() []EmailPart

	Attach(contentType string, data []byte) apperror.Error
	AttachReader(contentType string, reader io.ReadCloser) apperror.Error
	AttachFile(path string) apperror.Error

	GetAttachments() []EmailPart

	Embed(contentType string, data []byte) apperror.Error
	EmbedReader(contentType string, reader io.ReadCloser) apperror.Error
	EmbedFile(path string) apperror.Error

	GetEmbeddedAttachments() []EmailPart

	SetHeader(name string, values ...string)
	SetHeaders(map[string][]string)
}

/**
 * TemplateEngine.
 */

type TemplateEngine interface {
	Build(name string, tpl string) (interface{}, apperror.Error)
	BuildFile(name string, paths ...string) (interface{}, apperror.Error)

	GetTemplate(name string) interface{}

	BuildAndRender(name string, tpl string, data interface{}) ([]byte, apperror.Error)
	BuildFileAndRender(name string, data interface{}, paths ...string) ([]byte, apperror.Error)

	Render(name string, data interface{}) ([]byte, apperror.Error)

	// Clean up all templates.
	Clear()
}

/**
 * Method.
 */

type MethodHandler func(a App, r Request, unblock func()) Response

type Method interface {
	GetName() string
	IsBlocking() bool
	GetHandler() MethodHandler
}

/**
 * Resource.
 */

type Resource interface {
	Debug() bool
	SetDebug(bool)

	Dependencies() Dependencies
	SetDependencies(Dependencies)

	Backend() db.Backend
	SetBackend(db.Backend)

	IsPublic() bool

	Collection() string
	Model() Model
	SetModel(Model)
	CreateModel() Model

	Hooks() interface{}
	SetHooks(interface{})

	Q() db.Query

	Query(db.Query) ([]Model, apperror.Error)
	FindOne(id interface{}) (Model, apperror.Error)

	ApiFindOne(string, Request) Response
	ApiFind(db.Query, Request) Response

	Create(obj Model, user User) apperror.Error
	ApiCreate(obj Model, r Request) Response

	Update(obj Model, user User) apperror.Error
	// Updates the model by loading the current version from the database
	// and setting the changed values.
	PartialUpdate(obj Model, user User) apperror.Error

	ApiUpdate(obj Model, r Request) Response
	// See PartialUpdate.
	ApiPartialUpdate(obj Model, request Request) Response

	Delete(obj Model, user User) apperror.Error
	ApiDelete(id string, r Request) Response
}

/**
 * Generic service.
 */

type Service interface {
	SetDebug(bool)
	Debug() bool

	Dependencies() Dependencies
	SetDependencies(Dependencies)
}

/**
 * ResourceService.
 */

type ResourceService interface {
	Service

	Q(modelType string) (db.Query, apperror.Error)
	FindOne(modelType string, id string) (Model, apperror.Error)

	Create(Model, User) apperror.Error
	Update(Model, User) apperror.Error
	Delete(Model, User) apperror.Error
}

/**
 * FileService.
 */

type FileService interface {
	Service

	Resource() Resource
	SetResource(Resource)

	Backend(string) FileBackend
	AddBackend(FileBackend)

	DefaultBackend() FileBackend
	SetDefaultBackend(string)

	Model() interface{}
	SetModel(interface{})

	// Given a file instance with a specified bucket, read the file from filePath, upload it
	// to the backend and then store it in the database.
	// If no file.GetBackendName() is empty, the default backend will be used.
	// The file will be deleted if everything succeeds. Otherwise,
	// it will be left in the file system.
	// If deleteDir is true, the directory holding the file will be deleted
	// also.
	BuildFile(file File, user User, deleteDir bool) apperror.Error

	// Resource callthroughs.
	// The following methods map resource methods for convenience.

	// Create a new file model.
	New() File

	FindOne(id string) (File, apperror.Error)
	Find(db.Query) ([]File, apperror.Error)

	Create(File, User) apperror.Error
	Update(File, User) apperror.Error
	Delete(File, User) apperror.Error
}

/**
 * EmailService.
 */

type EmailService interface {
	Service

	SetDefaultFrom(EmailRecipient)

	Send(Email) apperror.Error
	SendMultiple(...Email) (apperror.Error, []apperror.Error)
}

/**
 * UserService.
 */

type UserService interface {
	Service

	AuthAdaptor(name string) AuthAdaptor
	AddAuthAdaptor(a AuthAdaptor)

	UserResource() Resource
	SetUserResource(Resource)

	ProfileResource() Resource
	SetProfileResource(resource Resource)

	ProfileModel() UserProfile

	SessionResource() Resource
	SetSessionResource(Resource)

	SetRoleResource(Resource)
	RoleResource() Resource

	SetPermissionResource(Resource)
	PermissionResource() Resource

	// Build a user token, persist it and return it.
	BuildToken(typ, userId string, expiresAt time.Time) (UserToken, apperror.Error)

	// Return a full user with roles and the profile joined.
	FindUser(userId interface{}) (User, apperror.Error)

	CreateUser(user User, adaptor string, data map[string]interface{}) apperror.Error
	AuthenticateUser(user User, adaptor string, data map[string]interface{}) (User, apperror.Error)
	VerifySession(token string) (User, Session, apperror.Error)

	SendConfirmationEmail(User) apperror.Error
	ConfirmEmail(token string) (User, apperror.Error)

	ChangePassword(user User, newPassword string) apperror.Error

	SendPasswordResetEmail(User) apperror.Error
	ResetPassword(token, newPassword string) (User, apperror.Error)
}

/**
 * Files.
 */

type BucketConfig interface {
}

// Interface for a File stored in a database backend.
type File interface {
	Model
	// File can belong to a user.
	UserModel

	// Retrieve the backend the file is stored in or should be stored in.
	// WARNING: can return nil.
	GetBackend() FileBackend
	SetBackend(FileBackend)

	GetBackendName() string
	SetBackendName(string)

	GetBackendID() string
	SetBackendID(string) error

	// File bucket.
	GetBucket() string
	SetBucket(string)

	GetTmpPath() string
	SetTmpPath(path string)

	// File name without extension.
	GetName() string
	SetName(string)

	// File extension if available.
	GetExtension() string
	SetExtension(string)

	// Name with extension.
	GetFullName() string
	SetFullName(string)

	GetTitle() string
	SetTitle(string)

	GetDescription() string
	SetDescription(string)

	// File size in bytes if available.
	GetSize() int64
	SetSize(int64)

	// Mime type if available.
	GetMime() string
	SetMime(string)

	GetIsImage() bool
	SetIsImage(bool)

	// File width and hight in pixels for images and videos.
	GetWidth() int
	SetWidth(int)

	GetHeight() int
	SetHeight(int)

	// Get a reader for the file.
	// Might return an error if the file does not exist in the backend,
	// or it is not connected to a backend.
	Reader() (io.ReadCloser, apperror.Error)

	// Get a writer for the file.
	// Might return an error if the file is not connected to a backend.
	Writer(create bool) (string, io.WriteCloser, apperror.Error)
}

type FileBackend interface {
	Name() string
	SetName(string)

	// Lists the buckets that currently exist.
	Buckets() ([]string, apperror.Error)

	// Check if a Bucket exists.
	HasBucket(string) (bool, apperror.Error)

	// Create a bucket.
	CreateBucket(string, BucketConfig) apperror.Error

	// Return the configuration for a a bucket.
	BucketConfig(string) BucketConfig

	// Change the configuration for a bucket.
	ConfigureBucket(string, BucketConfig) apperror.Error

	// Delete all files in a bucket.
	ClearBucket(bucket string) apperror.Error

	DeleteBucket(bucket string) apperror.Error

	// Clear all buckets.
	ClearAll() apperror.Error

	// Return the ids of all files in a bucket.
	FileIDs(bucket string) ([]string, apperror.Error)

	HasFile(File) (bool, apperror.Error)
	HasFileById(bucket, id string) (bool, apperror.Error)

	DeleteFile(File) apperror.Error
	DeleteFileById(bucket, id string) apperror.Error

	// Retrieve a reader for a file.
	Reader(File) (io.ReadCloser, apperror.Error)
	// Retrieve a reader for a file in a bucket.
	ReaderById(bucket, id string) (io.ReadCloser, apperror.Error)

	// Retrieve a writer for a file in a bucket.
	Writer(f File, create bool) (string, io.WriteCloser, apperror.Error)
	// Retrieve a writer for a file in a bucket.
	WriterById(bucket, id string, create bool) (string, io.WriteCloser, apperror.Error)
}

/**
 * Deps.
 */

type Dependencies interface {
	Logger() *logrus.Logger
	SetLogger(*logrus.Logger)

	Config() *config.Config
	SetConfig(*config.Config)

	Cache(name string) Cache
	Caches() map[string]Cache
	AddCache(cache Cache)
	SetCaches(map[string]Cache)

	DefaultBackend() db.Backend
	SetDefaultBackend(db.Backend)

	Backend(name string) db.Backend
	Backends() map[string]db.Backend
	AddBackend(b db.Backend)
	SetBackends(map[string]db.Backend)

	Resource(name string) Resource
	Resources() map[string]Resource
	AddResource(res Resource)
	SetResources(map[string]Resource)

	EmailService() EmailService
	SetEmailService(EmailService)

	FileService() FileService
	SetFileService(FileService)

	ResourceService() ResourceService
	SetResourceService(ResourceService)

	UserService() UserService
	SetUserService(UserService)

	TemplateEngine() TemplateEngine
	SetTemplateEngine(TemplateEngine)
}

/**
 * Frontend interfaces.
 */

type Frontend interface {
	Name() string

	App() App
	SetApp(App)

	Debug() bool
	SetDebug(bool)

	Logger() *logrus.Logger

	Init() apperror.Error
	Start() apperror.Error
}

/**
 * App interfaces.
 */

type App interface {
	ENV() string
	SetENV(string)

	Debug() bool
	SetDebug(bool)

	Dependencies() Dependencies

	Logger() *logrus.Logger
	SetLogger(*logrus.Logger)

	Config() *config.Config
	SetConfig(*config.Config)
	ReadConfig(path string)

	TmpDir() string

	Router() *httprouter.Router

	PrepareBackends()
	Run()
	RunCli()

	ServeFiles(route, path string)

	// Backend methods.

	RegisterBackend(backend db.Backend)
	Backend(name string) db.Backend
	DefaultBackend() db.Backend
	MigrateBackend(name string, version int, force bool) apperror.Error
	MigrateAllBackends(force bool) apperror.Error
	DropBackend(name string) apperror.Error
	DropAllBackends() apperror.Error
	RebuildBackend(name string) apperror.Error
	RebuildAllBackends() apperror.Error

	// Cache methods.

	RegisterCache(c Cache)
	Cache(name string) Cache

	// UserService methods.

	RegisterUserService(h UserService)
	UserService() UserService

	// FileService methods.

	RegisterFileService(f FileService)
	FileService() FileService

	// Email methods.

	RegisterEmailService(s EmailService)
	EmailService() EmailService

	// TemplateEngine methods.

	RegisterTemplateEngine(e TemplateEngine)
	TemplateEngine() TemplateEngine

	// Method methods.

	RegisterMethod(Method)
	RunMethod(name string, r Request, responder func(Response), withFinishedChannel bool) (chan bool, apperror.Error)

	// Resource methodds.

	RegisterResource(Resource)
	Resource(name string) Resource

	// Middleware methods.

	RegisterBeforeMiddleware(handler RequestHandler)
	ClearBeforeMiddlewares()
	BeforeMiddlewares() []RequestHandler

	RegisterAfterMiddleware(middleware AfterRequestMiddleware)
	ClearAfterMiddlewares()
	AfterMiddlewares() []AfterRequestMiddleware

	// Frontend methods.

	RegisterFrontend(Frontend)
	Frontend(name string) Frontend

	// HTTP related methods.

	NotFoundHandler() RequestHandler
	SetNotFoundHandler(x RequestHandler)

	RegisterHttpHandler(method, path string, handler RequestHandler)
}
