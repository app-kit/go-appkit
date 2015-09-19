package appkit

import (
	"io"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/olebedev/config"
	db "github.com/theduke/go-dukedb"
)

/**
 * User interfaces.
 */

type Session interface {
	db.Model

	SetType(string)
	GetType() string

	SetToken(string)
	GetToken() string

	SetUserID(string)
	GetUserID() string

	SetStartedAt(time.Time)
	GetStartedAt() time.Time

	SetValidUntil(time.Time)
	GetValidUntil() time.Time

	IsGuest() bool
}

type Role interface {
	GetName() string
	SetName(string)

	GetPermissions() []Permission
}

type Permission interface {
	GetName() string
	SetName(string)
}

type UserProfile interface {
	db.Model
}

type User interface {
	db.Model

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

	GetData() (interface{}, Error)
	SetData(interface{}) Error

	GetRoles() []Role
	AddRole(Role)
	RemoveRole(Role)
	ClearRoles()
	HasRole(Role) bool
	HasRoleStr(string) bool
}

type UserModel interface {
	db.Model
	GetUser() User
	SetUser(User)

	GetUserID() string
	SetUserID(string) error
}

type AuthItem interface {
	db.Model

	SetUserID(string)
	GetUserID() string

	SetType(string)
	GetType() string

	SetData(interface{}) Error
	GetData() (interface{}, Error)
}

type UserToken interface {
	db.Model
	GetType() string
	SetType(string)

	GetToken() string
	SetToken(string)

	GetUserID() string
	SetUserID(string) error

	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)
}

type AuthAdaptor interface {
	GetName() string

	BuildData(User, interface{}) (interface{}, Error)
	Authenticate(user User, authData interface{}, suppliedData interface{}) (bool, Error)
}

/**
 * Api interfaces.
 */

type Request interface {
	GetContext() *Context
	GetMeta() Context
	GetData() interface{}

	GetUser() User
	SetUser(User)

	GetSession() Session
	SetSession(Session)
}

type Response interface {
	GetError() Error

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
	GetKey() string
	SetKey(string)

	GetValue() interface{}
	SetValue(interface{})

	ToString() (string, Error)
	FromString(string) Error

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
	Set(CacheItem) Error
	SetString(key string, value string, expiresAt *time.Time, tags []string) Error

	// Retrieve a cache item from the cache.
	Get(key string, item ...CacheItem) (CacheItem, Error)
	GetString(key string) (string, Error)

	// Delete item from the cache.
	Delete(key ...string) Error

	// Get all keys stored in the cache.
	Keys() ([]string, Error)

	// Return all keys that have a certain tag.
	KeysByTags(tag ...string) ([]string, Error)

	// Clear all items from the cache.
	Clear() Error

	// Clear all items with the specified tags.
	ClearTag(tag string) Error

	// Clean up all expired entries.
	Cleanup() Error
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

	Attach(contentType string, data []byte) Error
	AttachReader(contentType string, reader io.ReadCloser) Error
	AttachFile(path string) Error

	GetAttachments() []EmailPart

	Embed(contentType string, data []byte) Error
	EmbedReader(contentType string, reader io.ReadCloser) Error
	EmbedFile(path string) Error

	GetEmbeddedAttachments() []EmailPart

	SetHeader(name string, values ...string)
	SetHeaders(map[string][]string)
}

/**
 * TemplateEngine.
 */

type TemplateEngine interface {
	Build(name string, tpl string) (interface{}, Error)
	BuildFile(name string, paths ...string) (interface{}, Error)

	Get(name string) interface{}

	BuildAndRender(name string, tpl string, data interface{}) ([]byte, Error)
	BuildFileAndRender(name string, data interface{}, paths ...string) ([]byte, Error)

	Render(name string, data interface{}) ([]byte, Error)

	// Clean up all templates.
	Clear()
}

/**
 * Method.
 */

type Method interface {
	Name() string
	IsBlocking() bool
	RequiresUser() bool
	Run(a App, r Request, unblock func()) Response
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

	Collection() string
	Model() db.Model
	SetModel(db.Model)
	NewModel() db.Model

	Hooks() interface{}
	SetHooks(interface{})

	Q() db.Query

	Find(db.Query) ([]db.Model, Error)
	FindOne(id string) (db.Model, Error)

	ApiFindOne(string, Request) Response
	ApiFind(db.Query, Request) Response
	// Same as find, but response meta will contain a total count.
	ApiFindPaginated(db.Query, Request) Response

	Create(obj db.Model, user User) Error
	ApiCreate(obj db.Model, r Request) Response

	Update(obj db.Model, user User) Error
	ApiUpdate(obj db.Model, r Request) Response

	Delete(obj db.Model, user User) Error
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

	Q(modelType string) (db.Query, Error)
	FindOne(modelType string, id string) (db.Model, Error)

	Create(db.Model, User) Error
	Update(db.Model, User) Error
	Delete(db.Model, User) Error
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
	BuildFile(file File, user User, filePath string, deleteDir bool) Error

	// Resource callthroughs.
	// The following methods map resource methods for convenience.

	// Create a new file model.
	New() File

	FindOne(id string) (File, Error)
	Find(db.Query) ([]File, Error)

	Create(File, User) Error
	Update(File, User) Error
	Delete(File, User) Error
}

/**
 * EmailService.
 */

type EmailService interface {
	Service

	SetDefaultFrom(EmailRecipient)

	Send(Email) Error
	SendMultiple(...Email) (Error, []Error)
}

/**
 * UserService.
 */

type UserService interface {
	Service

	CreateUser(user User, adaptor string, data interface{}) Error
	AuthenticateUser(user User, adaptor string, data interface{}) Error
	VerifySession(token string) (User, Session, Error)

	AuthAdaptor(name string) AuthAdaptor
	AddAuthAdaptor(a AuthAdaptor)

	UserResource() Resource
	SetUserResource(Resource)

	ProfileModel() UserProfile

	SessionResource() Resource
	SetSessionResource(Resource)

	SetRoleResource(Resource)
	RoleResource() Resource

	SetPermissionResource(Resource)
	PermissionResource() Resource
}

/**
 * Files.
 */

type BucketConfig interface {
}

// Interface for a File stored in a database backend.
type File interface {
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
	Reader() (io.ReadCloser, Error)

	// Get a writer for the file.
	// Might return an error if the file is not connected to a backend.
	Writer(create bool) (string, io.WriteCloser, Error)
}

type FileBackend interface {
	Name() string
	SetName(string)

	// Lists the buckets that currently exist.
	Buckets() ([]string, Error)

	// Check if a Bucket exists.
	HasBucket(string) (bool, Error)

	// Create a bucket.
	CreateBucket(string, BucketConfig) Error

	// Return the configuration for a a bucket.
	BucketConfig(string) BucketConfig

	// Change the configuration for a bucket.
	ConfigureBucket(string, BucketConfig) Error

	// Delete all files in a bucket.
	ClearBucket(bucket string) Error

	DeleteBucket(bucket string) Error

	// Clear all buckets.
	ClearAll() Error

	// Return the ids of all files in a bucket.
	FileIDs(bucket string) ([]string, Error)

	HasFile(File) (bool, Error)
	HasFileById(bucket, id string) (bool, Error)

	DeleteFile(File) Error
	DeleteFileById(bucket, id string) Error

	// Retrieve a reader for a file.
	Reader(File) (io.ReadCloser, Error)
	// Retrieve a reader for a file in a bucket.
	ReaderById(bucket, id string) (io.ReadCloser, Error)

	// Retrieve a writer for a file in a bucket.
	Writer(f File, create bool) (string, io.WriteCloser, Error)
	// Retrieve a writer for a file in a bucket.
	WriterById(bucket, id string, create bool) (string, io.WriteCloser, Error)
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

	ServeFiles(route, path string)

	RegisterBackend(backend db.Backend)
	Backend(name string) db.Backend
	MigrateBackend(name string, version int, force bool) Error
	MigrateAllBackends(force bool) Error
	DropBackend(name string) Error
	DropAllBackends() Error
	RebuildBackend(name string) Error
	RebuildAllBackends() Error

	RegisterCache(c Cache)
	Cache(name string) Cache

	RegisterEmailService(s EmailService)
	EmailService() EmailService

	RegisterTemplateEngine(e TemplateEngine)
	TemplateEngine() TemplateEngine

	RegisterMethod(Method)
	RunMethod(name string, r Request, responder func(Response), withFinishedChannel bool) (chan bool, Error)

	RegisterResource(Resource)
	Resource(name string) Resource

	RegisterUserService(h UserService)
	UserService() UserService

	RegisterFileService(f FileService)
	FileService() FileService

	RegisterBeforeMiddleware(handler RequestHandler)
	ClearBeforeMiddlewares()
	BeforeMiddlewares() []RequestHandler

	RegisterAfterMiddleware(middleware AfterRequestMiddleware)
	ClearAfterMiddlewares()
	AfterMiddlewares() []AfterRequestMiddleware

	NotFoundHandler() RequestHandler
	SetNotFoundHandler(x RequestHandler)

	RegisterHttpHandler(method, path string, handler RequestHandler)
}
