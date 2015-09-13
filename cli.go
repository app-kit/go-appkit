package appkit

import (
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

func (app *App) InitCli() {
	configPath := ""
	var cli = &cobra.Command{
		Use:   "",
		Short: "Start the server.",
		Long:  `Start the server`,

		Run: func(cmd *cobra.Command, args []string) {
			app.Run()
		},
	}
	cli.Flags().StringVarP(&configPath, "config", "c", "conf.yaml", "Config file in yaml format.")

	var migrateForce bool
	var migrateAll bool
	cmdMigrate := &cobra.Command{
		Use:   "migrate [backend] ([version])",
		Short: "Migrate a backend.",
		Long:  "Migrate a backend to newest or optionally specified version",
		Run: func(cmd *cobra.Command, args []string) {

			if migrateAll {
				if err := app.MigrateAllBackends(migrateForce); err != nil {
					log.Fatalf("Migration failed: %v", err)
				}
				return
			}

			if len(args) < 1 {
				log.Fatal("Usage: migrate backend [version]")
				return
			}

			backend := args[0]

			version := 0
			if len(args) == 2 {
				var err error
				version, err = strconv.Atoi(args[1])
				if err != nil {
					log.Fatal("Version must be a number")
					return
				}
			}

			if err := app.MigrateBackend(backend, version, migrateForce); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}

			log.Printf("Migrations succeded")
		},
	}
	cmdMigrate.Flags().BoolVarP(&migrateForce, "force", "f", false, "Force migration on locked backend")
	cmdMigrate.Flags().BoolVarP(&migrateAll, "all", "a", false, "Migrate all backends to newest version")
	cli.AddCommand(cmdMigrate)

	var rebuildAll bool
	cmdDbRebuild := &cobra.Command{
		Use:   "db-rebuild [backend]",
		Short: "Rebuild a backend by dropping all collections and running migrations.",
		Long:  `Rebuild a backend by dropping all collections and running migrations`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				if rebuildAll {
					app.RebuildAllBackends()
					return
				} else {
					log.Printf("Usage: db-rebuild BACKEND")
					return
				}
			}

			app.RebuildBackend(args[0])
		},
	}
	cmdDbRebuild.Flags().BoolVarP(&rebuildAll, "all", "a", false, "Rebuild all backends")
	cli.AddCommand(cmdDbRebuild)

	var dropAll bool
	cmdDbDrop := &cobra.Command{
		Use:   "db-drop [backend]",
		Short: "Drop all collections from a backend.",
		Long:  `Drop all collections from a backend`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				if dropAll {
					app.DropAllBackends()
					return
				} else {
					log.Printf("Need to specify backend.")
				}
			}

			app.DropBackend(args[0])
		},
	}
	cmdDbDrop.Flags().BoolVarP(&dropAll, "all", "a", false, "Drop all backends")
	cli.AddCommand(cmdDbDrop)

	app.Cli = cli
}

func (app *App) RunCli() {
	app.Cli.Execute()
}
