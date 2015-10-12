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
