# Appkit

This project aims to provide an application framework for developing 
web applications and APIs in the GO language.

The endgoal is to provide a complete framework similar to [Meteor](https://www.meteor.com/),
but with an efficient and compiled language in the backend.

**Main features:**

* [DukeDB ORM](https://github.com/theduke/go-dukedb) supporting different databases (PostgreSQL, MySQL,  MongoDB, ...)
* Different frontends ([JSONAPI](http://jsonapi.org/), [WAMP](http://wamp-proto.org/) (under development).
* Full user and permission system with password and OAUTH authentication (easily extendable), *roles* and *permissions*.
* Server side rendering of javascript apps (Ember, AngularJS, ...) with PhantomJS.
* Easily extendable CLI.
* File storage with different backends (File system included, easily extendable to Amazon S3 etc).
* Caching system with different caches (File system, in memory and REDIS included, easily extendable).
* [Scaffolding CLI](https://github.com/theduke/go-appkit-cli) similar to Yeoman for quick setup and development.
* **Optional** light weight CMS with menu system and pages with an Admin frontend written in EmberJS.


## TOC

1. [Concepts](https://github.com/theduke/go-appkit#Concepts)
  * [Models](https://github.com/theduke/go-appkit#Concepts.Models)
  * [Resources](https://github.com/theduke/go-appkit#Concepts.Resources)
  * [Methods](https://github.com/theduke/go-appkit#Concepts.Methods)
  * [User system](https://github.com/theduke/go-appkit#Concepts.Usersystem)
  * [File storage](https://github.com/theduke/go-appkit#Concepts.Filestorage)
2. [Getting started](https://github.com/theduke/go-appkit#Gettingstarted)
  * [Minimal Todo](https://github.com/theduke/go-appkit#Gettingstarted.Minimaltodo)
  * [Todo with Usersystem](https://github.com/theduke/go-appkit#Gettingstarted.TodoWithUsers)
3. [Documentation](https://github.com/theduke/go-appkit#docs)
  * [Resources](https://github.com/theduke/go-appkit#docs.resources)
4. [Additional Information](https://github.com/theduke/go-appkit#additional)

<a name="Concepts"></a>
## Concepts

<a name="Concepts.Models"></a>
### Models

The API revolves about models which are just GO structs.

For Appkit to understand your models, your structs need to implement a few interfaces.

* Collection() string: return a name for your model collection. Eg "todos" for your 'Todo' struct.
GetID() interface{}: Return the id
SetID(id interface{}) error: Set the ID. Return an error if the given ID is invalid or nil otherwise.
GetStrID() string: Return a string version of the ID. Empty string if no ID is set yet.
SetStrID(id string) error: Set the ID from a string version of the ID. Return error if given ID is invalid, or nil otherwise.

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

You can find all available hooks in the [Resources documentation](https://github.com/theduke/go-appkit#docs.resources)

There are also several supplied resource implementations for common use cases.

<a name="Concepts.Methods"></a>
### Methods

<a name="Concepts.Usersystem"></a>
### User system

<a name="Concepts.Filestorage"></a>
### File storage


<a name="Gettingstarted"></a>
## Getting started

You should first read over the *Models*, *Resources* and *Methods* section in [Concepts](https://github.com/theduke/go-appkit#Concepts), and 
then check out the [Todo example](https://github.com/theduke/go-appkit#Gettingstarted.Minimaltodo) to familiarize yourself with the way Appkit works.

After that, run these  commands to **create a new Appkit project:**
```bash
go get github.com/theduke/go-appkitcli
go install github.com/theduke/go-appkitcli/appkit

appkit bootstrap --backend="postgres" myproject

cd myproject/myproject

go run main.go
```

### Examples

The examples use a **non-persistent in memory backend**.

You can use all backends supported by [DukeDB](https://github.com/theduke/go-dukedb) (the recommended one is PostgreSQL).

To use a different backend, refer to the [Backends section](https://github.com/theduke/go-appkit#Backends).

<a name="Gettingstarted.Minimaltodo"></a>
#### Minimal Todo Example

The following example shows how to create a very simple todo application, where projects and todos can be created by users without an account. 

To see how to employ the user system, refer to the next section.

Save this code into a file "todo.go" or just download the [file](https://github.com/theduke/go-appkit/tree/master/examples/todo-minimal.go)

```go
package main

import(
	"time"

	"github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/backends/memory"
	"github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app"
	"github.com/theduke/go-appkit/resources"
)

type Project struct {
	// IntIDModel contains an ID uint64 field and some methods implementing the appkit.Model interface.
	// You can also implemnt the methods yourself.
	// For details, refer to the [Concepts](https://github.com/theduke/go-appkit#Concepts.Models) and the DukeDB documentation.
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

You just can embed the *UserModel* base struct in your models, and alter the resources registration to use the  *users.UserResource* mixin.

By doing that, your project and todo models with belong to a user, and create, update and delete operations will be  restricted to admins and owners of the model.

Save this code into a file "todo.go" or just download the [file](https://github.com/theduke/go-appkit/tree/master/examples/todo-users.go).

```go
package main

import (
	"time"

	"github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/backends/memory"

	"github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/users"
)

type Project struct {
	// IntIDModel contains an ID uint64 field and some methods implementing the appkit.Model interface.
	// You can also implemnt the methods yourself.
	// For details, refer to the [Concepts](https://github.com/theduke/go-appkit#Concepts.Models) and the DukeDB documentation.
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
	app.RegisterResource(resources.NewResource(&Project{}, &users.UserResource{}, true))
	app.RegisterResource(resources.NewResource(&Todo{}, &users.UserResource{}, true))

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

* [Resource implementations](https://github.com/theduke/go-appkit#docs.resources.implementations)
* [Hooks](https://github.com/theduke/go-appkit#docs.resources.hooks)

<a name="docs.resources.implementations"></a>
#### Resource implementations

The package contains several resource implementations that fulfill common needs, making it unneccessary to implement the  hooks yourself.

* [ReadOnlyResource](https://github.com/theduke/go-appkit#docs.resources.implementations.readonly)
* [AdminResource](https://github.com/theduke/go-appkit#docs.resources.implementations.admin)
* [LoggedInResource](https://github.com/theduke/go-appkit#docs.resources.implementations.loggedin)
* [PublicWriteResource](https://github.com/theduke/go-appkit#docs.resources.implementations.publicwrite)
* [UserResource](https://github.com/theduke/go-appkit#docs.resources.implementations.user)

<a name="docs.resources.implementations.readonly"></a>
##### ReadOnlyResource

This resource only allows READ operations via the API, no create, update or delete.

```go
import(
  ...
  "github.com/theduke/go-appkit/resources"
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
  "github.com/theduke/go-appkit/resources"
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
  "github.com/theduke/go-appkit/resources"
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
  "github.com/theduke/go-appkit/resources"
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
  "github.com/theduke/go-appkit/resources"
  "github.com/theduke/go-appkit/users"
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

* General
  * HttpRoutes
  * Methods
* Find
  * AllowFind
  * ApiFindOne
  * ApiFind
  * ApiAlterQuery
  * ApiAfterFind
* Create
  * Create
  * ApiCreate
  * BeforeCreate
  * AllowCreate
  * AfterCreate
* Update
  * ApiUpdate
  * Update
  * BeforeUpdate
  * AllowUpdate
  * AfterUpdate
* Delete
  * ApiDelete
  * Delete
  * BeforeDelete
  * AllowDelete
  * AfterDelete

<a name="docs.resources.hooks.general"></a>
##### General

###### HttpRoutes

```go
HttpRoutes(kit.Resource)(kit.Resource) []kit.HttpRoute
```
Supply http route connected with your resource

##### Methods

```go
Methods(kit.Resource) []kit.Method
```

Supply methods connected with your resource (See [Methods](https://github.com/theduke/go-appkit#Concepts.Methods)).


<a name="docs.resources.find"></a>
#### Find

##### AllowFind

```go
AllowFind(res kit.Resource, model kit.Model, user kit.User) bool
```

Restrict what users may retrieve a model

##### ApiFindOne

```go
ApiFindOne(res kit.Resource, rawId string, r kit.Request) kit.Response
```

Overwrite the FindOne behaviour.

##### ApiFind

```go
ApiFind(res kit.Resource, query db.Query, r kit.Request) kit.Response
```

Overwrite the Find behaviour.

##### ApiAlterQuery

```go
ApiAterQuery(res kit.Resource, query db.Query, r kit.Request) apperror.Error
```

Alter a find query before it is executed. For example to restrict fields based on the users permissions.

##### ApiAfterFind

```go
ApiAfterFind(res kit.Resource, obj []kit.Model, user kit.User) apperror.Error
```

Execute code after find, for example to alter model data.


<a name="docs.resources.create"></a>
#### Create

##### ApiCreate

```go
ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
```

Overwrite the ApiCreate behaviour.

#####  Create

```go
Create(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the default Create behaviour.

#####  BeforeCreate

```go
BeforeCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code before creating a model. Allows to abort creation by returning an error.

#####  AllowCreate

```go
AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool
```

Access control for creation, for example to restrict creation to certain user roles.

#####  AfterCreate

```go
AfterCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code after creation, for example to create related models.


<a name="docs.resources.update"></a>
#### Update

#####  ApiUpdate

```go
ApiUpdate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
```

Overwrite the ApiUpdate behaviour.

#####  Update

```go
Update(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the Update behaviour.

#####  BeforeUpdate

```go
BeforeUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
```

Run code before update. Allows to abort update by returning an error.

##### AllowUpdate

```go
AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool
```

Restrict update operations, for example to restrict updates to the models owner or admins.

#####  AfterUpdate

```go
AfterUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
```

Run code after updates.


<a name="docs.resources.delete"></a>
#### Delete

#####  ApiDelete

```go
ApiDelete(res kit.Resource, id string, r kit.Request) kit.Response
```

Overwrite te ApiDelete behaviour.

##### Delete

```go
Delete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Overwrite the Delete behaviour.

#####  BeforeDelete

```go
BeforeDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
```

Run code before deleting. Allows to abort deletion by returning an error.

#####  AllowDelete

```go
AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool
```

Restrict delete operations. For example to only allow admins to delete.

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
