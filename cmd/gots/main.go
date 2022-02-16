package main

import (
	"flag"
	"io/fs"
	"log"

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

	var www fs.FS
	var err error
	if path == "" {
		www = gots.Source
	} else {
		// www = os.DirFS(path)
		www, err = code.NewLocalFS(path)
		if err != nil {
			log.Fatal(err)
		}
	}
	app := code.UI(www, ui.Mode("app"))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
