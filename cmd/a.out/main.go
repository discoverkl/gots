package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/discoverkl/gots/ui"
)

func sum(a, b int) int {
	return a + b
}

func timer(ctx context.Context, write *ui.Function) string {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return "cancel"
		case <-time.After(time.Millisecond * 100):
			v := write.Call(i)
			if v.Err() != nil {
				log.Printf("timer callback call error: %v", v.Err())
			}
		}
	}
	return "done"
}

type Counter struct {
	sum int
}

func newCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Add() int {
	c.sum++
	return c.sum
}

func main() {
	app := ui.New(
		ui.Root(os.DirFS("fe/dist")),
		ui.OnlinePort(8000),
		ui.BlurOnClose(true),
		ui.HistoryMode(true),
		// ui.OnlinePrefix("/me"),
		// ui.OnlineAttach(ui.HTTPServerFunc(http.Handle), false),
		// ui.LocalExitDelay(5 * time.Second),
		// ui.OnlineAuth(ui.BasicAuth(func(user, pass string) bool {
		// 	return user == "admin" && pass == "123"
		// })),
		// ui.OnlineTLS("server.crt", "server.key"),
	)

	app.BindFunc("sum", sum)
	app.BindFunc("timer", timer)

	// app.BindFunc("math.pow", math.Pow)
	// app.BindFunc("math.abs", math.Abs)
	// app.BindPrefix("utils.time", ui.Map(map[string]interface{}{"timer": timer}))
	// app.BindPrefix("counter", ui.DelayObject(&Counter{}, func(c *ui.UIContext) ui.Bindings {
	// 	go func() {
	// 		<-c.Done
	// 		log.Println("page done")
	// 	}()
	// 	return ui.Object(newCounter())
	// }))

	// app2 := ui.New(ui.OnlinePort(8001))
	// app2.BindFunc("getEnv", os.Getenv)
	// // go app2.Run()
	// app.Add("hi", app2)

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "This is a normal Go server")
	// })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	// if err := http.ListenAndServe(":8000", nil); err != nil {
	// 	log.Fatal(err)
	// }
}
