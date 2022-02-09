package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/discoverkl/gots"
	"github.com/discoverkl/gots/code"
	"github.com/discoverkl/gots/ui"
)

func main() {
	var path string

	flag.Parse()

	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	var www http.FileSystem
	if path == "" {
		www = http.FS(gots.Source)
	} else {
		www = http.Dir(path)
	}
	app := code.UI(www, ui.Mode("app"))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
