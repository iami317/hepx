package frpc

import (
	"embed"

	"github.com/iami317/hepx/assets"
)

//go:embed static/*
var content embed.FS

func init() {
	assets.Register(content)
}
