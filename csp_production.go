//go:build production

package main

import "github.com/wailsapp/wails/v2/pkg/options/assetserver"

// wails build always adds the production tag (also with -debug), and Wails
// refuses to start a binary built with neither dev nor production, so every
// runnable non-dev binary gets the CSP.
var assetMiddleware assetserver.Middleware = withCSP
