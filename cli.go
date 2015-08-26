package appkit

import (
	"github.com/spf13/cobra"
)

func (app *App) InitCli() {
	configPath := ""
	var cli = &cobra.Command{
		Use:   "Server",
		Short: "Start the server.",
		Long:  `Start the server`,

		Run: func(cmd *cobra.Command, args []string) {
			app.ReadConfig(configPath)
			app.Run()
		},
	}
	cli.Flags().StringVarP(&configPath, "config", "c", "conf.yaml", "Config file in yaml format.")

	app.Cli = cli
}

func (app *App) RunCli() {
	app.Cli.Execute()
}
