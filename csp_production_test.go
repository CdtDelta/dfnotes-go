//go:build production

package main

import "testing"

func TestProductionBuildInstallsCSP(t *testing.T) {
	if assetMiddleware == nil {
		t.Fatal("production build has no CSP middleware")
	}
}
