package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/discoverkl/gots/code"
)

func main() {
	var path string

	flag.Parse()

	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	app := code.UI(http.Dir(path))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
