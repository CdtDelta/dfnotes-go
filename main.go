package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func createMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Export Case", nil, func(_ *menu.CallbackData) {
		app.handleExportCaseMenu()
	})
	fileMenu.AddText("Settings", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:settings")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Lock Case", keys.CmdOrCtrl("l"), func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:lock-case")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		wailsruntime.Quit(app.ctx)
	})

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Theme", nil, func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:theme")
	})

	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("User Guide", nil, func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:user-guide")
	})
	helpMenu.AddText("About", nil, func(_ *menu.CallbackData) {
		wailsruntime.MessageDialog(app.ctx, wailsruntime.MessageDialogOptions{
			Type:    wailsruntime.InfoDialog,
			Title:   "About DFNotes",
			Message: "DFNotes - Digital Forensic Notebook\nSecure, tamper-evident case notes.",
		})
	})

	return appMenu
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "dfnotes-go",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Menu:             createMenu(app),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
