package static

import (
	"embed"
)

//go:embed css/*
var Css embed.FS

//go:embed img/*
var Img embed.FS

//go:embed js/*
var Js embed.FS

//go:embed views/*
var Views embed.FS
