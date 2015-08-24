package main

import (
	"log"


	//"github.com/theduke/appkit"
	kitgorm "github.com/theduke/appkit/backends/gorm"
	"github.com/theduke/appkit/users"
	"github.com/theduke/appkit/app"
)

func start() error {
	// Build backend.
	backend, err := kitgorm.NewBackend("postgres://theduke:theduke@localhost/docduke?sslmode=disable")	
	if err != nil {
		return err
	}

	userHandler := users.NewUserHandler()
	//userResource := userHandler.GetUserResource()

	app := app.NewApp("")	
	app.RegisterBackend("gorm", backend)
	app.RegisterUserHandler(userHandler)

	app.Run()

	return nil
}

func main() {
	err := start()
	log.Printf("error: %v\n", err)
}
