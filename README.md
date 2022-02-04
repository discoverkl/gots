# Gots

Call remote Go methods directly from TypeScript through WebSocket.

# Helloword

Try yourself:
```shell
git clone https://github.com/discoverkl/gots-examples.git
cd gots-examples/helloworld
go run .
```

Source code:

- [main.go](https://github.com/discoverkl/gots-examples/blob/main/helloworld/main.go)

```go
package main

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/discoverkl/gots/ui"
)

//go:embed fe/dist
var root embed.FS

func add(a, b int) int {
	return a + b
}

func main() {
	www, _ := fs.Sub(root, "fe/dist")
	app := ui.New(
		ui.Mode(promptRunMod()),
		ui.Root(http.FS(www)),
		ui.OnlineAddr(":8000"),
	)
	app.BindFunc("add", add)
	app.Run()
}

func promptRunMod() string {
	for {
		fmt.Print(promptText)
		ch, _, err := bufio.NewReader(os.Stdin).ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println()
				os.Exit(0)
			}
			log.Fatal(err)
		}

		switch ch {
		case '1':
			return "page"
		case '2':
			return "app"
		case '3':
			return "online"
		case 'q':
		default:
			os.Exit(0)
		}
	}
}

const promptText = `
*** Commands ***

1: LocalPage - start a local web server, open its' serving url with your default web browser
2: LocalApp - start a local web server, open its' serving url within a native app (which is a chrome process)
3: Online   - run a online web server

Please enter (1-3)? `
```

- [index.html](https://github.com/discoverkl/gots-examples/blob/main/helloworld/fe/dist/index.html)

```html
<!DOCTYPE html>

<html>
    <head><title>Go and TypeScript</title></head>
    <body>
        <h1>Hello World!</h1>

        1 + 3 = <span id="sum"></span>

        <!-- Step 1: export Gots to window object -->
        <script src="https://cdn.jsdelivr.net/npm/ts2go@1.0.0/ts2go.js"></script>

        <script>
          async function main() {
            // Step 2: create a rpc client
            const api = await Gots.getapi()

            // Step 3: call remote go function: add
            document.getElementById("sum").innerText = await api.add(1, 3)
          }
          main()
        </script>
    </body>

</html>
```

- [go.mod](https://github.com/discoverkl/gots-examples/blob/main/helloworld/go.mod)

```go-module
module github.com/discoverkl/gots-examples/helloworld

go 1.17

require github.com/discoverkl/gots v0.1.2

require golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
```
