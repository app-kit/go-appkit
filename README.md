# Appkit

This project aims to provide an application framework for developing 
web applications and APIs in the GO language.

The endgoal is to provide a complete framework similar to [Meteor](https://www.meteor.com/),
but with an efficient and compiled language in the backend.

## Warning

**This project is still in a very early development phase and not ready for production use.**

**Main features:**

* [DukeDB ORM](https://github.com/theduke/go-dukedb) supporting different databases (PostgreSQL, MySQL,  MongoDB, ...) and with a *migrations system*.
* Different frontends REST, [JSONAPI](http://jsonapi.org/), websockets with [WAMP](http://wamp-proto.org/) (under development).
* Arbitrary **client side queries with full power of DukeDB**.
* Subscribe to model updates, insertions and deletions on the client with **PubSub** with WAMP or long polling. *Still under development*.
* Full user system with user registration, password reset, notification emails...
* User *Authentication* (password and OAUTH included, easily extendable).
* User *Authorization*: RBAC system with roles and permissions.
* Easy to use **server side rendering of javascript apps** (Ember, AngularJS, ...) with PhantomJS.
* Easily extendable CLI.
* File storage with different backends (File system included, easily extendable to Amazon S3 etc).
* Caching system with different caches (File system, in memory and REDIS included, easily extendable).
* [Scaffolding CLI](https://github.com/app-kit/go-appkitcli) similar to Yeoman for quick setup and development.
* **Optional** light weight CMS with menu system and pages with an Admin frontend written in EmberJS.
* Ember CLI addon for easy integration into the [Ember JS framework](emberjs.com).

## TOC

1. [Concepts](https://github.com/app-kit/go-appkit#Concepts)
  * [Frontends](https://github.com/app-kit/go-appkit#Concepts.Frontends)
  * [Models](https://github.com/app-kit/go-appkit#Concepts.Models)
  * [Resources](https://github.com/app-kit/go-appkit#Concepts.Resources)
  * [Methods](https://github.com/app-kit/go-appkit#Concepts.Methods)
  * [DukeDB, backends and client side queries](https://github.com/app-kit/go-appkit#Concepts.dukedb)
  * [User system](https://github.com/app-kit/go-appkit#Concepts.Usersystem)
  * [File storage](https://github.com/app-kit/go-appkit#Concepts.Filestorage)
  * [Server side rendering](https://github.com/app-kit/go-appkit#Concepts.serversiderendering)
  * [Caching](https://github.com/app-kit/go-appkit#Concepts.caching)
  * [Registry and Services](https://github.com/app-kit/go-appkit#Concepts.registry)
2. [Getting started](https://github.com/app-kit/go-appkit#Gettingstarted)
  * [Setup](https://github.com/app-kit/go-appkit#Gettingstarted.setup)
  * [Example: Minimal Todo](https://github.com/app-kit/go-appkit#Gettingstarted.Minimaltodo)
  * [Example: Todo with Usersystem](https://github.com/app-kit/go-appkit#Gettingstarted.TodoWithUsers)
3. [Documentation](https://github.com/app-kit/go-appkit#docs)
  * [Resources](https://github.com/app-kit/go-appkit#docs.resources)
4. [Additional Information](https://github.com/app-kit/go-appkit#additional)

<a name="Concepts"></a>
## Concepts

<a name="Concepts.Frontends"></a>
### Frontends

The API can be accessed through various frontends, and you can quite easily implement your own if the available ones do not fit your requirements.

By default, when you start your Appkit server, you will have a REST frontend and a [JSONAPI](http://jsonapi.org) frontend. Soon, there will also be support for websockets via the [WAMP protocol](http://wamp-proto.org/).

#### REST

With the rest frontend, you can access the API with simple HTTP calls.

* POST */api/method/query*: Query for resources.
* POST */api/method/create*: Create a model.
* POST */api/method/update*: Update a model.
* POST */api/method/delete*: Delete a model.

* POST */api/method/my.method*: Your custom methods.

#### JSONAPI

[JSONAPI](http://jsonapi.org) is a specification for a CRUD api with well defined support for 
relationships.

Ember uses JSONAPI by default starting with version 1.13.

* GET */api/users*: Query for resources.
* POST */api/users*: Create a new model.
* PATCH */api/users/ID*: Update a model
* DELETE */api/users/ID*: Delete a model

Other methods can be accessed in the same way as specified for REST.

#### Websockets via WAMP

[WAMP](http://wamp-proto.org/) is a websocket sub-protocol that supports RPC and PubSub.

You can use it inside your browser apps with [Autobahn JS](http://autobahn.ws/js/).

Wamp is a powerful protocol that allows fast and efficient communication with the api,
and has very nice support for PubSub which enables efficient live updates on the client.

**WAMP support is still under development**.


<a name="Concepts.Models"></a>
### Models

The API revolves about models which are just GO structs.

For Appkit to understand your models, your structs need to implement a few interfaces.

* Collection() string: return a name for your model collection. Eg "todos" for your 'Todo' struct.
* GetID() interface{}: Return the id
* SetID(id interface{}) error: Set the ID. Return an error if the given ID is invalid or nil otherwise.
* GetStrID() string: Return a string version of the ID. Empty string if no ID is set yet.
* SetStrID(id string) error: Set the ID from a string version of the ID. Return error if given ID is invalid, or nil otherwise.

DukeDB offers embeddable base structs that implement all interfaces except collection: *dukedb.IntIDModel* if your models use an integer ID or *dukedb.StrIDModel* for models with a string ID (like MongoDB uses).

```go
type Todo struct {
  dukedb.IntIDModel
  
  Name string
  ...
}

func (Todo) Collection() string {
  return "todos"
}
```

<a name="Concepts.Resources"></a>
### Resources

Your models are exposed via the API in the form of resources.
Each resources is tied to a model, and has an optional resource struct that controls the behaviour of your resource.

To register a resource with your app:
```go
type Todo struct {
  ...
}

type TodoResource struct {}

app.RegisterResource(&Todo{}, &TodoResource)
```


There are many hooks you can implement on your resource to control behaviour, for example to restrict access or to run code before or after creation, deletion, etc.

You can also alter the default CRUD operations by implementing some of these hooks.

There are also several supplied resource implementations for common use cases.

You can find more information in the [Resources documentation](https://github.com/app-kit/go-appkit#docs.resources)


<a name="Concepts.Methods"></a>
### Methods

All API operations that do not correspond to simple CRUD operations are exposed in the form of methods.

Methods can be registered directly from your resources with the [Methods() hook](https://github.com/app-kit/go-appkit#docs.resources.hooks.methods), or with the app.

Methods can be *blocking* or *non-blocking*.

When a method is blocking, all other method call by the same user 
will wait until the blocking method is finished. 

This might be neccessary when you, for example, create a new model and then
retrieve a list of models.
When the create method blocks, the list method will only run once the 
creation has finished, and will therefore include the new model. 

Example of a simple method that returns the count of a certain model.
```go
import(
	"github.com/app-kit/go-appkit/api/methods"
)

countMethod := &methods.Method{
	Name: "todos.count",
	Blocking: false,
	Handler: func(a kit.App, r kit.Request, unblock func()) kit.Response {
		count, err := a.Resource("todos").Q().Count()
		if err != nil {
			return appkit.NewErrorResponse("db_error", err)
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"count": count},
		}
	},
}

app.RegisterMethod(countMethod)
```

The method is now available with:
```
GET localhost:8000/method/todos.count

# Response:
{
	data: {
		count: 210
	}
}
```

<a name="Concepts.dukedb"></a>
### DukeDB, backends and client side queries

Appkit uses the [DukeDB ORM](https://github.com/theduke/go-dukedb) which allows to use your database of choice.

The recommended and currently best supported database is PostgreSQL (also MySQL), with MongoDB support coming soon.

#### Backends

Appkit makes it easy to mix several backends.

For example, you could store most of your models in PostgreSQL, 
but the sessions in memory.

To make a resource use a certain backend:
```go
backend := app.Registry().Backend("memory")
app.Resource("sessions").SetBackend(backend)
```

#### Client side queries

A great feature are client side queries, which allow you to use the full power of ORM queries right from your client, without writing any server side logic.

For example, you can do:

```
GET /api/todos?query=xxx

# Query is serialized json:
{
  limit: 20,
  offset: 10,
  filters: {
    intField: {$gt: 25},
   type: {$in: ["a",  "b"}
  },
  joins: ["realtionA"],
  fields: ["fieldA", "fieldB", "relationA.fieldX"],
  order: ["fieldA", "-relationA.fieldX"]
}
```

The filters support all [MongoDB style query operators](http://docs.mongodb.org/manual/reference/operator/query/).

<a name="Concepts.Usersystem"></a>
### User system

Appkit comes with a full-fledged user system that supports user signup, password reset, email activation, **authentication** and **authorization**.

You can easily create users from the client with simple api calls.

#### Authentication

Once a user is signed up, the client can login and create a session with simple REST or RPC calls.

Once a session token is retrieved, all subsequent requests must add a *Authorization: token* header that identifies the client.

#### Authorization

The auth system is based on [RBAC](https://en.wikipedia.org/wiki/Role-based_access_control).

You can create roles assign permissions to roles.
Each user can have multiple roles.

In your server logic, you can easily check if the current user has a certain role or permission, 
allowing easy access control.

#### User management

The system comes with support for welcome mails, email confirmation, and password resets.

<a name="Concepts.Filestorage"></a>
### File storage

Appkit comes with a system for storing uploaded files.

The system is *storage agnostic*.
A filesystem storage is used by default, but you can easily implement your own storage solution that could, for example, use Amazon S3.

Files information is also stored in the database, and you can easily implement models that have files attached to them.

Files can be either public with no access control, or restricted access based on user roles/permissions.

#### Serving files

Files can be accessed via the http route:
```
GET /files/ID/file-name.txt
```

#### Serving images/thumbnails

Appkit also comes with a system for generating thumbnails or applying some filters to images.

To serve an image scaled to a width and height, and a grayscale filter applied, use:
```
GET /images/ID/file-name?width=500&height=200&filters=grayscale
```

<a name="Concepts.serversiderendering"></a>
### Server side rendering

A common annoyance with modern javascript web applications is the lack of support for delivering fully rendered responses, since all rendering is done in the browser.

This also makes it hard to do SEO, since the crawlers can not properly crawl your website, and can not identify removed pages since no 404 responses can be delivered.

Appkit allows you to very easily enable server side rendering, which will render your application on the server by using [PhantomJS](http://phantomjs.org).

The response will then be *cached* and fully delivered to the client.
You can let your frontend application take over control after the first user interaction (eg. click on a link).

To enable server side rendering, add this section to your config.yaml:
```yaml
frontend:
  indexTpl: public/index.html
serverRenderer:
  enabled: true
  cache: fs
  cacheLifetime: 3600
```

On the client side, inside your app, you have to report once the rendering of the route has finished.
This way you can also set the HTTP status code.

All you have to do, once the page is fully rendered (by using, for example, your frontend routers afterRender hook):
```javascript
window.serverRenderer = {
	status: 200
};
```

<a name="Concepts.caching"></a>
### Caching

Appkit comes with a caching system that supports various caches.

If you do not want to use the included ones, it is easy to implement your own cache.

The included ones are:
* Filesystem (cache entries are stored in files on disk)
* Memory (in memory cache)
* **[Redis](http://redis.io)** (recommended!)


<a name="Concepts.registry"></a>
### Registry and Services

The registry gives you access to all parts of your application.
It can be accessed within your methods and resources.

The functionality is split into services, which must implement the respective interface.

This gives you the power to implement your own service if the default does not fit your needs.

* `app.Registry().DefaultBackend() | returns dukedb.Backend`
* `app.Registry().Backend("postgres") | returns dukedb.Backend`

* `app.Registry().Resource("todos") | returns appkit.Resource`

* `app.Registry().UserService() | returns appkit.UserServvice`
* `app.Registry().FileService() | returns appkit.FileService`
* `app.Registry().EmailService() | returns appkit.EmailService`

* `app.Registry().DefaultCache() | returns appkit.Cache`
* `app.Registry().Cache("fs") | returns appkit.Cache`

* `app.Registry().TemplateEngine() | returns appkit.TemplateEngine`

* `app.Registry().Config() | returns appkit.Config`

* `app.Registry().Logger() | returns *logrus.Logger`


<a name="Gettingstarted"></a>
## Getting started


<a name="Gettingstarted.setup"></a>
### Setup

You should first read over the *Models*, *Resources* and *Methods* section in [Concepts](https://github.com/app-kit/go-appkit#Concepts), and 
then check out the [Todo example](https://github.com/app-kit/go-appkit#Gettingstarted.Minimaltodo) to familiarize yourself with the way Appkit works.

After that, run these  commands to **create a new Appkit project:**
```bash
go get github.com/app-kit/go-appkitcli
go install github.com/app-kit/go-appkitcli/appkit

appkit bootstrap --backend="postgres" myproject

cd myproject/myproject

go run main.go
```

### Examples

The examples use a **non-persistent in memory backend**.

You can use all backends supported by [DukeDB](https://github.com/theduke/go-dukedb) (the recommended one is PostgreSQL).

To use a different backend, refer to the [Backends section](https://github.com/app-kit/go-appkit#Backends).

<a name="Gettingstarted.Minimaltodo"></a>
#### Minimal Todo Example

The following example shows how to create a very simple todo application, where projects and todos can be created by users without an account. 

To see how to employ the user system, refer to the next section.

Save this code into a file "todo.go" or just download the [file](https://github.com/app-kit/go-appkit/tree/master/examples/todo-minimal.go)

```go
package main

import(
	"time"

	"github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/backends/memory"
	"github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/app"
	"github.com/app-kit/go-appkit/resources"
)

type Project struct {
	// IntIDModel contains an ID uint64 field and some methods implementing the appkit.Model interface.
	// You can also implemnt the methods yourself.
	// For details, refer to the [Concepts](https://github.com/app-kit/go-appkit#Concepts.Models) and the DukeDB documentation.
	dukedb.IntIDModel

	Name string `db:"not-null;max:100"`
	Description string `db:"max:5000"`
}

func (Project) Collection() string {
	return "projects"
}

type Todo struct {
	dukedb.IntIDModel

	Project *Project
	ProjectID uint64 `db:"not-null"`

	Name string `db:"not-null;max:300"`
	Description string `db:"max:5000"`
	DueDate time.Time
	FinishedAt *time.Time
}

func (Todo) Collection() string {
	return "todos"
}

func BuildApp() appkit.App {
	app := app.NewApp()
	
	// Set up memory backend.
	backend := memory.New()
	app.RegisterBackend(backend)

	// Set up resources.
	app.RegisterResource(resources.NewResource(&Project{}, &resources.PublicWriteResource{}, true))
	app.RegisterResource(resources.NewResource(&Todo{}, &resources.PublicWriteResource{}, true))

	return app
}

func main() {
	app := BuildApp()
	app.RunCli()
}
```

**That's it.** 

You now have a working CLI that can launch a server with a [JSONAPI](http://jsonapi.org/) frontend (on localhost:8000 by default).

After starting the server, you can perform CRUD operations for projects and todos.

##### Run the server:
`go run todo.go`

##### Create a new project.

```
POST http://localhost:8000/api/projects
-----------------------------------
{
	data: {
    attributes: {
			name: "My First Project",
			description: "Project description"
  	}
  }
}

# Response:
{
	data: {
		type: "projects",
		id: 1,
		attributes: ....
	}
}
```

##### Create a new todo:

```
POST http://localhost:8000/api/todos
-----------------------------------
{
	data: {
    attributes: {
			name: "Todo 1",
			description: "Some todo",
			dueDate: "2015-10-11"
  	},
  	relationships: {
  		project: {
  			type: "projects",
  			id: 1
  		}
  	}
  }
}
```

##### Find all projects.

```
GET localhost:8000/api/projects
```

##### Find all todos of a project.

```
GET localhost:8000/api/todos?filters=projectId:1
```

##### Set todo as finished.

```
POST http://localhost:8000/api/todos/1
-----------------------------------
{
	data: {
    attributes: {
			finishedAt: "2015-10-11T17:53:03Z",
  	}
  }
}
```



<a name="Gettingstarted.TodoWithUsers"></a>
#### Todo with user system

This example is largely equivalent to the previous one, but it employs Appkit's user system
by tying projects and todos to users.

**The changes required are minimal.**

You just can embed the *UserModel* base struct in your models, and alter the resources registration to use the  *resources.UserResource* mixin.

By doing that, your project and todo models with belong to a user, and create, update and delete operations will be  restricted to admins and owners of the model.

Save this code into a file "todo.go" or just download the [file](https://github.com/app-kit/go-appkit/tree/master/examples/todo-users.go).

```go
package main

import (
	"time"

	"github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/backends/memory"

	"github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/app"
	"github.com/app-kit/go-appkit/resources"
	"github.com/app-kit/go-appkit/users"
)

type Project struct {
	// IntIDModel contains an ID uint64 field and some methods implementing the appkit.Model interface.
	// You can also implemnt the methods yourself.
	// For details, refer to the [Concepts](https://github.com/app-kit/go-appkit#Concepts.Models) and the DukeDB documentation.
	dukedb.IntIDModel

	users.IntUserModel

	Name        string `db:"not-null;max:100"`
	Description string `db:"max:5000"`
}

func (Project) Collection() string {
	return "projects"
}

type Todo struct {
	dukedb.IntIDModel

	users.IntUserModel

	Project   *Project
	ProjectID uint64 `db:"not-null"`

	Name        string `db:"not-null;max:300"`
	Description string `db:"max:5000"`
	DueDate     time.Time
	FinishedAt  *time.Time
}

func (Todo) Collection() string {
	return "todos"
}

func BuildApp() appkit.App {
	app := app.NewApp()

	// Set up memory backend.
	backend := memory.New()
	app.RegisterBackend(backend)

	// Set up resources.
	app.RegisterResource(resources.NewResource(&Project{}, &resources.UserResource{}, true))
	app.RegisterResource(resources.NewResource(&Todo{}, &resources.UserResource{}, true))

	return app
}

func main() {
	app := BuildApp()
	app.RunCli()
}
```

Before you can create and update projects and todos, you need to create a user.

After that, you must create a session for the user, which will give you an auth token that you must supply
in the 'Authentication:' header.

The authentication system allows for different authentication adapters.

The default is a password adaptor.

##### Create a user.

```
POST http://localhost:8000/users
-----------------------------------
{
	data: {
    attributes: {
			email: "user1@gmail.com"
  	}
  },
  meta: {
  	adaptor: "password",
  	"auth-data": {
  		"password": "my password"
  	}
  }
}

```

##### Log  in by creating a session.

```
POST http://localhost:8000/sessions
-----------------------------------
{
	data: {},
  meta: {
  	user: "user1@gmail.com",
  	adaptor: "password",
  	"auth-data": {
  		"password": "my password"
  	}
  }
}

# Response:
...
	token: "xxxxxxxxxx"
...
```

##### CRUD operations.

Now that you have a user and a session token, you can start creating projects and todos like before.
All you need to do is add an `Authentication: my_token` header to the requests and use the requests 
from  the previous example one to one.


<a name="docs"></a>
## Documentation

<a name="docs.resources"></a>
### Resources

* [Resource implementations](https://github.com/app-kit/go-appkit#docs.resources.implementations)
* [Hooks](https://github.com/app-kit/go-appkit#docs.resources.hooks)

<a name="docs.resources.implementations"></a>
#### Resource implementations

The package contains several resource implementations that fulfill common needs, making it unneccessary to implement the  hooks yourself.

* [ReadOnlyResource](https://github.com/app-kit/go-appkit#docs.resources.implementations.readonly)
* [AdminResource](https://github.com/app-kit/go-appkit#docs.resources.implementations.admin)
* [LoggedInResource](https://github.com/app-kit/go-appkit#docs.resources.implementations.loggedin)
* [PublicWriteResource](https://github.com/app-kit/go-appkit#docs.resources.implementations.publicwrite)
* [UserResource](https://github.com/app-kit/go-appkit#docs.resources.implementations.user)

<a name="docs.resources.implementations.readonly"></a>
##### ReadOnlyResource

This resource only allows READ operations via the API, no create, update or delete.

```go
import(
  ...
  "github.com/app-kit/go-appkit/resources"
  ...
)

app.RegisterResource(&Model{}, &resources.ReadOnlyResource{})
```

<a name="docs.resources.implementations.admin"></a>
##### AdminResource

This resource restricts create, read and update operations to users with the 'admin' role, or with the permission 'collectionname.create/update/delete'.

```go
import(
  ...
  "github.com/app-kit/go-appkit/resources"
  ...
)

app.RegisterResource(&Model{}, &resources.AdminResource{})
```

<a name="docs.resources.implementations.loggedin"></a>
##### LoggedInResource

This resource restricts create, read and update operations to logged in users.
**This is the default behaviour used if you do not supply your own resource struct.**

```go
import(
  ...
  "github.com/app-kit/go-appkit/resources"
  ...
)

app.RegisterResource(&Model{}, &resources.LoggedInResource{})
```

<a name="docs.resources.implementations.publicwrite"></a>
##### PublicWriteResource

This resource allows all create/update/delete operations for all api users, even without authentication.

```go
import(
  ...
  "github.com/app-kit/go-appkit/resources"
  ...
)

app.RegisterResource(&Model{}, &resources.PublicWriteResource{})
```

<a name="docs.resources.implementations.user"></a>
##### UserResource

This resource restricts create, read and update operations to **users that OWN a model**, have the admin role, or the 'collection.create/read/update' permission.

For this to work, your model has to implement the *appkit.UserModel* interface.

**This is the default behaviour used if you do not supply your own resource struct.**

```go
import(
  ...
  "github.com/app-kit/go-appkit/resources"
  "github.com/app-kit/go-appkit/users"
  ...
)

type Model struct {
  dukedb.IntIDModel
  users.IntUserModel
}

app.RegisterResource(&Model{}, &resources.UserResource{})
```

<a name="docs.resources.hooks"></a>
#### Hooks

Here you can find all the available hooks you can implement on your resources.

* [General](https://github.com/app-kit/go-appkit#docs.resources.hooks.general)
  * [HttpRoutes](https://github.com/app-kit/go-appkit#docs.resources.hooks.httproutes)
  * [Methods](https://github.com/app-kit/go-appkit#docs.resources.hooks.methods)
* [Find](https://github.com/app-kit/go-appkit#docs.resources.hooks.find)
  * [AllowFind](https://github.com/app-kit/go-appkit#docs.resources.hooks.allowfind)
  * [ApiFindOne](https://github.com/app-kit/go-appkit#docs.resources.hooks.apifindone)
  * [ApiFind](https://github.com/app-kit/go-appkit#docs.resources.hooks.apifind)
  * [ApiAlterQuery](https://github.com/app-kit/go-appkit#docs.resources.hooks.apialterquery)
  * [ApiAfterFind](https://github.com/app-kit/go-appkit#docs.resources.hooks.apiafterfind)
* [Create](https://github.com/app-kit/go-appkit#docs.resources.hooks.createoverview)
  * [Create](https://github.com/app-kit/go-appkit#docs.resources.hooks.create)
  * [ApiCreate](https://github.com/app-kit/go-appkit#docs.resources.hooks.apicreate)
  * [BeforeCreate](https://github.com/app-kit/go-appkit#docs.resources.hooks.beforecreate)
  * [AllowCreate](https://github.com/app-kit/go-appkit#docs.resources.hooks.allowcreate)
  * [AfterCreate](https://github.com/app-kit/go-appkit#docs.resources.hooks.aftercreate)
* [Update](https://github.com/app-kit/go-appkit#docs.resources.hooks.updateoverview)
  * [ApiUpdate](https://github.com/app-kit/go-appkit#docs.resources.hooks.apiupdate)
  * [Update](https://github.com/app-kit/go-appkit#docs.resources.hooks.update)
  * [BeforeUpdate](https://github.com/app-kit/go-appkit#docs.resources.hooks.beforeupdate)
  * [AllowUpdate](https://github.com/app-kit/go-appkit#docs.resources.hooks.allowupdate)
  * [AfterUpdate](https://github.com/app-kit/go-appkit#docs.resources.hooks.afterupdate)
* [Delete](https://github.com/app-kit/go-appkit#docs.resources.hooks.deleteoverview)
  * [ApiDelete](https://github.com/app-kit/go-appkit#docs.resources.hooks.apidelete)
  * [Delete](https://github.com/app-kit/go-appkit#docs.resources.hooks.delete)
  * [BeforeDelete](https://github.com/app-kit/go-appkit#docs.resources.hooks.beforedelete)
  * [AllowDelete](https://github.com/app-kit/go-appkit#docs.resources.hooks.allowdelete)
  * [AfterDelete](https://github.com/app-kit/go-appkit#docs.resources.hooks.afterdelete)

<a name="docs.resources.hooks.general"></a>
##### General

<a name="docs.resources.hooks.httproutes"></a>
###### HttpRoutes

```go
HttpRoutes(kit.Resource)(kit.Resource) []kit.HttpRoute
```
Supply http route connected with your resource

<a name="docs.resources.hooks.methods"></a>
###### Methods

```go
Methods(kit.Resource) []kit.Method
```

Supply methods connected with your resource (See [Methods](https://github.com/app-kit/go-appkit#Concepts.Methods)).


<a name="docs.resources.hooks.find"></a>
##### Find

<a name="docs.resources.hooks.allowfind"></a>
###### AllowFind

```go
AllowFind(res kit.Resource, model kit.Model, user kit.User) bool
```

Restrict what users may retrieve a model

<a name="docs.resources.hooks.apifindone"></a>
###### ApiFindOne

```go
ApiFindOne(res kit.Resource, rawId string, r kit.Request) kit.Response
```

Overwrite the FindOne behaviour.

<a name="docs.resources.hooks.apifind"></a>
###### ApiFind

```go
ApiFind(res kit.Resource, query db.Query, r kit.Request) kit.Response
```

Overwrite the Find behaviour.

<a name="docs.resources.hooks.apialterquery"></a>
###### ApiAlterQuery

```go
ApiAterQuery(res kit.Resource, query db.Query, r kit.Request) apperror.Error
```

Alter a find query before it is executed. For example to restrict fields based on the users permissions.

<a name="docs.resources.hooks.apiafterfind"></a>
###### ApiAfterFind

```go
ApiAfterFind(res kit.Resource, obj []kit.Model, user kit.User) apperror.Error
```

Execute code after find, for example to alter model data.


<a name="docs.resources.createoverview"></a>
##### Create

<a name="docs.resources.hooks.apicreate"></a>
###### ApiCreate

```go
ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
```

Overwrite the ApiCreate behaviour.

<a name="docs.resources.hooks.create"></a>
###### Create

```go
Create(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the default Create behaviour.

<a name="docs.resources.hooks.beforecreate"></a>
###### BeforeCreate

```go
BeforeCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code before creating a model. Allows to abort creation by returning an error.

<a name="docs.resources.hooks.allowcreate"></a>
######  AllowCreate

```go
AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool
```

Access control for creation, for example to restrict creation to certain user roles.

<a name="docs.resources.hooks.aftercreate"></a>
######  AfterCreate

```go
AfterCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code after creation, for example to create related models.


<a name="docs.resources.updateoverview"></a>
##### Update

<a name="docs.resources.hooks.apiupdate"></a>
######  ApiUpdate

```go
ApiUpdate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
```

Overwrite the ApiUpdate behaviour.

<a name="docs.resources.hooks.update"></a>
######  Update

```go
Update(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the Update behaviour.

<a name="docs.resources.hooks.beforeupdate"></a>
###### BeforeUpdate

```go
BeforeUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
```

Run code before update. Allows to abort update by returning an error.

<a name="docs.resources.hooks.allowupdate"></a>
###### AllowUpdate

```go
AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool
```

Restrict update operations, for example to restrict updates to the models owner or admins.

<a name="docs.resources.hooks.afterupdate"></a>
###### AfterUpdate

```go
AfterUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
```

Run code after updates.


<a name="docs.resources.hooks.deleteoverview"></a>
##### Delete

<a name="docs.resources.hooks.apidelete"></a>
######  ApiDelete

```go
ApiDelete(res kit.Resource, id string, r kit.Request) kit.Response
```

Overwrite te ApiDelete behaviour.

<a name="docs.resources.hooks.delete"></a>
###### Delete

```go
Delete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the Delete behaviour.

<a name="docs.resources.hooks.beforedelete"></a>
###### BeforeDelete

```go
BeforeDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code before deleting. Allows to abort deletion by returning an error.

<a name="docs.resources.hooks.allowdelete"></a>
######  AllowDelete

```go
AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool
```

Restrict delete operations. For example to only allow admins to delete.

<a name="docs.resources.hooks.afterdelete"></a>
#####  AfterDelete

```go
AfterDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code after deletion, for example to clean up related resources.


<a name="additional"></a>
## Additional Information

### Warning

This project is still under heavy development.

*Use with caution.*

### Changelog

https://raw.githubusercontent.com/theduke/go-appkit/master/CHANGELOG.txt


### Versioning

This project uses [SEMVER](http://semver.org).

All compatability breaking changes will result in a new version.

Respective versions can be found in the respository branch.

### Contributing

All contributions are highly welcome.

Just create an issue or a pull request on Github.

### License

This project is under the MIT License.

For Details, see LICENSE.txt
