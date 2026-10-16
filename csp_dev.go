//go:build !production

package main

import "github.com/wailsapp/wails/v2/pkg/options/assetserver"

// No CSP under wails dev: Wails still applies Middleware when proxying to the
// Vite dev server, and Vite's inline React refresh preamble and HMR websocket
// would be blocked.
var assetMiddleware assetserver.Middleware
