package main

import (
	"net/http"
	"strings"
)

// contentSecurityPolicy is defense in depth against script injection through
// rendered note HTML. It is sent as a response header rather than a meta tag
// because the header applies before the document parser runs (a meta tag let
// the WebKitGTK preload scanner fetch a remote <img> in testing) and because
// frame-ancestors is ignored in a meta tag.
//
// script-src must never include 'unsafe-inline' or 'unsafe-eval'; Wails loads
// its runtime and IPC shims as same-origin <script src> tags, and Go-to-JS
// calls (events, binding callbacks) run through the native WebView evaluate
// API, which page CSP does not govern.
var contentSecurityPolicy = strings.Join([]string{
	"default-src 'none'",
	"script-src 'self'",
	// 'unsafe-inline' keeps style="" attributes in existing raw-HTML notes
	// working until rendering stops passing raw HTML through.
	"style-src 'self' 'unsafe-inline'",
	// data: carries attachment images; blob: is unused today but harmless
	// because blob URLs can only be minted by script that already runs here.
	"img-src 'self' data: blob:",
	"font-src 'self' data:",
	"connect-src 'self'",
	"object-src 'none'",
	"base-uri 'none'",
	"form-action 'none'",
	"frame-src 'none'",
	"frame-ancestors 'none'",
}, "; ")

// withCSP sets the Content-Security-Policy header on every AssetServer
// response, including the index.html that Wails rewrites to inject its
// runtime scripts.
func withCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}
