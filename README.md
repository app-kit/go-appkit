# Appkit

This project aims to provide an application framework for developing 
web applications and APIs in the GO language.

The endgoal is to provide a complete framework similar to [Meteor](https://www.meteor.com/),
but with an efficient and compiled language in the backend.

**Main features:**

* [DukeDB ORM](https://github.com/theduke/go-dukedb) supporting different databases (PostgreSQL, MySQL,  MongoDB, ...)
* Different frontends ([JSONAPI](http://jsonapi.org/), [WAMP](http://wamp-proto.org/) (under development).
* Full user system with password and OAUTH authentication (easily extendable).
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

<a name="Concepts"></a>
## Concepts

<a name="Concepts.Models"></a>
### Models

The API revolves about models which are just GO structs.

For Appkit to understand your models, your structs need to implement a few interfaces.

<a name="Concepts.Resources"></a>
### Resources

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



## Additional Information

### Warning

This project is still under heavy development.

*Use with caution.*

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
