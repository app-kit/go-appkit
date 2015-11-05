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
	// IntIdModel contains an Id uint64 field and some methods implementing the appkit.Model interface.
	// You can also implemnt the methods yourself.
	// For details, refer to the [Concepts](https://github.com/app-kit/go-appkit#Concepts.Models) and the DukeDB documentation.
	dukedb.IntIdModel

	users.IntUserModel

	Name        string `db:"required;max:100"`
	Description string `db:"max:5000"`
}

func (Project) Collection() string {
	return "projects"
}

type Todo struct {
	dukedb.IntIdModel

	users.IntUserModel

	Project   *Project
	ProjectId uint64 `db:"required"`

	Name        string `db:"required;max:300"`
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
