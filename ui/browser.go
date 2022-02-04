package ui

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

type NativeWindow interface {
	Open(url string) error
	Close()
}

type browserNativeWindow struct{}

func (*browserNativeWindow) Open(url string) error {
	openBrowser(url)
	return nil
}

func (*browserNativeWindow) Close() {}

type browserPage struct {
	server    *FileServer
	closeOnce sync.Once
	done      chan struct{}
	win       NativeWindow
	mapURL    func(net.Listener) string
}

func NewPage(root http.FileSystem) Window {
	return NewNativeWindow(root, &browserNativeWindow{}, nil)
}

func NewPageMapURL(root http.FileSystem, mapURL func(net.Listener) string) Window {
	return NewNativeWindow(root, &browserNativeWindow{}, mapURL)
}

func NewNativeWindow(root http.FileSystem, win NativeWindow, mapURL func(net.Listener) string) Window {
	return &browserPage{
		server: NewFileServer(root),
		done:   make(chan struct{}),
		win:    win,
		mapURL: mapURL,
	}
}

func (c *browserPage) Bind(b Bindings) error {
	return c.server.Bind(b)
}

func (c *browserPage) Server() *FileServer {
	return c.server
}

func (c *browserPage) Open() error {
	// ** local server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	addr := listener.Addr().(*net.TCPAddr)
	log.Println("using port:", addr.Port)

	c.server.Listener = listener
	go c.server.ListenAndServe()

	var url string
	url = fmt.Sprintf("http://localhost:%d", addr.Port)
	if c.mapURL != nil {
		url = c.mapURL(listener)
	}

	// ** brower page
	if err := c.win.Open(url); err != nil {
		return err
	}

	// 1/2 server.Done() => done
	// 2/2 user call Close() => done
	<-c.server.Done()
	c.Close()
	return nil
}

func (c *browserPage) SetExitDelay(d time.Duration) {
	c.server.localServerExitDelay = d
}

func (c *browserPage) Eval(js string) Value {
	panic("Not Implemented")
}

func (c *browserPage) Done() <-chan struct{} {
	return c.done
}

func (c *browserPage) Close() error {
	c.closeOnce.Do(func() {
		if dev {
			log.Println("Window.Close called")
		}
		c.server.Close()
		<-c.server.Done()
		if dev {
			log.Println("Window.server done")
		}

		c.win.Close()

		// notify finally close
		close(c.done)
	})
	return nil
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}
