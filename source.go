package gots

import "embed"

//go:embed homedir one ui
//go:embed cmd/gots/*.go
//go:embed code/*.go code/fe/index.html code/fe/src
//go:embed go.mod *.go README.md
var Source embed.FS
