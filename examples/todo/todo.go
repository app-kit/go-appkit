package main

import (
	"log"
	"time"
	"strconv"


	//"github.com/theduke/go-appkit"
	_ "github.com/lib/pq"
	"github.com/jinzhu/gorm"

	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
	dbgorm "github.com/theduke/go-dukedb/backends/gorm"
	"github.com/theduke/go-appkit/users"
)

type Project struct {
	ID uint64 `gorm:"primary_key"`
	Name string

	Todos []*Todo

	Todo *Todo
	TodoID uint64
}

func (b Project) GetID() string {
	return strconv.FormatUint(b.ID, 10)
}

func (b Project) SetID(rawId string) error {
	id, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

func (p Project) GetCollection() string {
	return "projects"
}

type ProjectHooks struct {
}

func (p ProjectHooks) BeforeCreate(res kit.ApiResource, obj db.Model, user kit.ApiUser) kit.ApiError {
	log.Printf("obj: %+v\n", obj)
	return nil
}

type Todo struct {
	ID uint64 `gorm:"primary_key"`

	Name string
	Comments string
	DueDate time.Time
	Priority int

	Project *Project
	ProjectID uint64
}

func (b Todo) GetID() string {
	return strconv.FormatUint(b.ID, 10)
}

func (b Todo) SetID(rawId string) error {
	id, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

func (t Todo) GetCollection() string {
	return "todos"
}

func InitMigrations(app *kit.App) {
	handler := app.GetBackend("gorm").(db.MigrationBackend).GetMigrationHandler()
	
	userMigrations := users.GetUserMigrations(app)
	handler.Add(userMigrations[0])
	handler.Add(userMigrations[1])


	v2 := db.Migration{
		Name: "create tables",
		Up: func(b db.MigrationBackend) error {
			db := b.(*dbgorm.Backend).Db
			if err := db.CreateTable(&Todo{}).Error; err != nil {
				return err
			}
			if err := db.CreateTable(&Project{}).Error; err != nil {
				return err
			}

			return nil
		},
	}
	handler.Add(v2)
}

func start() error {
	// Build backend.
	db, err := gorm.Open("postgres", "user=theduke dbname=docduke sslmode=disable")
	if err != nil {
		return err
	}

	backend := dbgorm.New(&db)
	backend.SetDebug(true)

	userHandler := users.NewUserHandler()
	//userResource := userHandler.GetUserResource()

	app := kit.NewApp("")	
	app.RegisterBackend("gorm", backend)

	app.RegisterUserHandler(userHandler)
	app.RegisterResource(&Project{}, ProjectHooks{})
	app.RegisterResource(&Todo{}, nil)

	app.PrepareBackends()

	app.RegisterMethod(&kit.Method{
		Name: "todo-count",
		RequiresUser: false,
		Run: func(app *kit.App, request *kit.Request) (interface{}, kit.ApiError) {
			todos := app.GetResource("projects")
			count, _ := todos.GetQuery().Last()

			return map[string]interface{}{
				"count": count,
			}, nil
		},
	})

	InitMigrations(app)

	app.RunCli()

	return nil
}

func main() {
	err := start()
	log.Printf("error: %v\n", err)
}

