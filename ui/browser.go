package ui

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

type browserPage struct {
	server    *FileServer
	closeOnce sync.Once
	done      chan struct{}
	openURL   func(string) error
	mapURL    func(net.Listener) string
}

func NewPage(root fs.FS, openURL func(string) error) Window {
	return NewPageMapURL(root, openURL, nil)
}

func NewPageMapURL(root fs.FS, openURL func(string) error, mapURL func(net.Listener) string) Window {
	return &browserPage{
		server:  NewFileServer(root),
		done:    make(chan struct{}),
		openURL: openURL,
		mapURL:  mapURL,
	}
}

func (c *browserPage) OpenURL(url string) error {
	var err error
	if c.openURL == nil {
		// default to open system browser
		err = OpenBrowser(url)
	} else {
		err = c.openURL(url) // this call could block
	}
	if err != nil {
		err = fmt.Errorf("open url failed: %w", err)
		log.Fatal(err)
	}
	<-c.server.Done()
	return err
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
	if err := c.OpenURL(url); err != nil {
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

		c.Close()

		// notify finally close
		close(c.done)
	})
	return nil
}

func OpenBrowser(pageURL string) error {
	link, err := url.Parse(pageURL)
	if err != nil {
		return err
	}
	url := link.String()
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "android":
		var path string
		path, err = exec.LookPath("termux-open-url")
		if err != nil {
			err = fmt.Errorf("can't find command termux-open-url")
		} else {
			err = exec.Command(path, url).Start()
		}
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
