package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/ad201/deephermes/app"
	"github.com/ad201/deephermes/pkg/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	cliMode := flag.Bool("cli", false, "Run in CLI mode (no GUI)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *cliMode {
		runCLI(cfg)
		return
	}

	// Desktop mode
	desktopApp := app.NewApp(cfg)

	err = wails.Run(&options.App{
		Title:     "DeepHermes",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     desktopApp.OnStartup,
		OnShutdown:    desktopApp.OnShutdown,
		OnBeforeClose: desktopApp.OnBeforeClose,
		Bind: []interface{}{
			desktopApp,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "deephermes-desktop",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				desktopApp.RestoreMainWindow()
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
