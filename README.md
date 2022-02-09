# Gots

Call remote Go methods directly from TypeScript through WebSocket.

# Helloword

Try yourself:
```shell
go run github.com/discoverkl/gots-examples/helloworld@v0.1.3
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
	"log"
	"net/http"
	"os"

	"github.com/discoverkl/gots/ui"
)

//go:embed index.html
var www embed.FS

func add(a, b int) int {
	return a + b
}

func main() {
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
		case '1', 10:
			return "app"
		case '2':
			return "page"
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

1: LocalApp - start a local web server, open its' serving url within a native app
2: LocalPage - start a local web server, open its' serving url with your default web browser
3: Online   - run a online web server

Please enter (1-3)? `
```

- [index.html](https://github.com/discoverkl/gots-examples/blob/main/helloworld/index.html)

```html
<!DOCTYPE html>

<html>
    <head><title>Go and TypeScript</title></head>
    <body>
        <h1>Hello World!</h1>

        <p style="font-size: 40px;" id="sum"></span>

        <!-- Step 1: export Gots to window object -->
        <script src="https://cdn.jsdelivr.net/npm/ts2go@1.0.0/ts2go.js"></script>

        <script>
          function rand() {
            return Math.floor(Math.random() * 100);
          }
          async function main() {
            // Step 2: create a rpc client
            const api = await Gots.getapi()

            // Step 3: call remote go function: add
            let a = rand(), b = rand();
            document.getElementById("sum").innerText = a + " + " + b + " = " + await api.add(a, b)
          }
          main()
        </script>
    </body>

</html>
```
