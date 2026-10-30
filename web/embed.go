// Package web embeds the built Quasar app (npm run build writes dist/spa).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func Dist() fs.FS {
	sub, _ := fs.Sub(dist, "dist/spa")
	return sub
}
