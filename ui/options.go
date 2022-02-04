package ui

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Option func(*uiConfig) error

type uiConfig struct {
	Mode            string
	Quiet           bool
	BlurOnClose     bool
	HistoryMode     bool
	Root            http.FileSystem
	AppX            int
	AppY            int
	AppWidth        int
	AppHeight       int
	AppChromeArgs   []string
	AppChromeBinary string
	OnlineAddr      string
	OnlineListener  net.Listener
	OnlinePrefix    string
	OnlineAuth      func(http.HandlerFunc) http.HandlerFunc
	OnlineCertFile  string
	OnlineKeyFile   string
	OnlineAttach    HTTPServer
	OnlineAttachTLS bool
	LocalMapURL     func(net.Listener) string
	LocalExitDelay  *time.Duration
}

func defaultUIConfig() *uiConfig {
	return &uiConfig{
		Root:        defaultRoot,
		BlurOnClose: true,
		AppX:        200,
		AppY:        200,
		AppWidth:    1024,
		AppHeight:   768,
	}
}

//
// General Options
//

// Mode can be override by a 'MODE=xxx' environment variable.
// Default value is page.
func Mode(mod string) Option {
	return func(c *uiConfig) error {
		switch mod {
		case "", modApp, modPage, modOnline:
			c.Mode = mod
		default:
			return fmt.Errorf("invalid run mode: %v", mod)
		}
		return nil
	}
}

func Quiet() Option {
	return func(c *uiConfig) error {
		c.Quiet = true
		return nil
	}
}

func BlurOnClose(blur bool) Option {
	return func(c *uiConfig) error {
		c.BlurOnClose = blur
		return nil
	}
}

//
// FileSystem Options
//

func HistoryMode(enable bool) Option {
	return func(c *uiConfig) error {
		c.HistoryMode = enable
		return nil
	}
}

func Root(root http.FileSystem) Option {
	return func(c *uiConfig) error {
		c.Root = root
		return nil
	}
}

func RootHtml(html string) Option {
	return func(c *uiConfig) error {
		c.Root = htmlRoot(html)
		return nil
	}
}

func RootFiles(files map[string]string) Option {
	return func(c *uiConfig) error {
		c.Root = NewSimpleRoot(files)
		return nil
	}
}

//
// App Mode Options
//

func AppWindow(x, y, width, height int) Option {
	return func(c *uiConfig) error {
		c.AppX, c.AppY = x, y
		c.AppWidth, c.AppHeight = width, height
		return nil
	}
}

func AppFullScreen() Option {
	return AppChromeArgs("--window-position=0,0", "--window-size=5000,5000", "--start-fullscreen")
}

func AppChromeArgs(args ...string) Option {
	return func(c *uiConfig) error {
		c.AppChromeArgs = append(c.AppChromeArgs, args...)
		return nil
	}
}

func AppChromeBinary(path string) Option {
	return func(c *uiConfig) error {
		c.AppChromeBinary = path
		return nil
	}
}

//
// Page Mode Options
//

//
// Online Mode Options
//

func OnlinePort(port int) Option {
	return func(c *uiConfig) error {
		c.OnlineAddr = fmt.Sprintf(":%d", port)
		return nil
	}
}

func OnlineAddr(addr string) Option {
	return func(c *uiConfig) error {
		c.OnlineAddr = addr
		return nil
	}
}

func OnlineListener(listener net.Listener) Option {
	return func(c *uiConfig) error {
		c.OnlineListener = listener
		return nil
	}
}

func OnlinePrefix(prefix string) Option {
	return func(c *uiConfig) error {
		c.OnlinePrefix = prefix
		return nil
	}
}

func OnlineAuth(auth func(http.HandlerFunc) http.HandlerFunc) Option {
	return func(c *uiConfig) error {
		c.OnlineAuth = auth
		return nil
	}
}

func OnlineTLS(certFile, keyFile string) Option {
	return func(c *uiConfig) error {
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("cert/key file is empty")
		}
		c.OnlineCertFile = certFile
		c.OnlineKeyFile = keyFile
		return nil
	}

}

func OnlineAttach(existingServer HTTPServer, tls bool) Option {
	return func(c *uiConfig) error {
		c.OnlineAttach = existingServer
		c.OnlineAttachTLS = tls
		return nil
	}
}

//
// Local Options
//

func LocalMapURL(mapURL func(net.Listener) string) Option {
	return func(c *uiConfig) error {
		c.LocalMapURL = mapURL
		return nil
	}
}

// LocalExitDelay contorl auto exit behaviour of a local server.
// By default, when all clients where lost, a local server will wait for a new client and exit if timeout.
// A local server will exit immediately after any client lost when duration is 0,
// and never exit when duration is less than 0.
func LocalExitDelay(d time.Duration) Option {
	return func(c *uiConfig) error {
		c.LocalExitDelay = &d
		return nil
	}
}

//
// Simple Http FileSystem
//

const defaultRoot = htmlRoot(`<!DOCTYPE html>
<html>
	<head>
		<title>Universal User Interface</title>
	</head>
    <body>
        <h1>Hello, Go and TypeScript!</h1>
        <script src="/gots.js?name=window"></script>
    </body>
</html>
`)

type htmlRoot string

func (r htmlRoot) Open(name string) (http.File, error) {
	return NewHtmlRoot(string(r)).Open(name)
}

type simpleRoot struct {
	files     map[string]string
	listCache []*stringFile
	once      sync.Once
}

func NewHtmlRoot(html string) *simpleRoot {
	files := map[string]string{"index.html": html}
	return NewSimpleRoot(files)
}

func NewSimpleRoot(files map[string]string) *simpleRoot {
	return &simpleRoot{
		files: files,
		once:  sync.Once{},
	}
}

func (r *simpleRoot) Open(name string) (http.File, error) {
	r.once.Do(func() {
		cache := make([]*stringFile, 0, len(r.files))
		badNames := []string{}
		for name, text := range r.files {
			if index := strings.LastIndex(name, "/"); index != -1 && index != 0 {
				log.Printf("name should not contain '/': %s", name)
			}
			absName := name
			if name == "" || name[0] != '/' {
				absName = "/" + name
				badNames = append(badNames, name)
			}
			r.files[absName] = text
			cache = append(cache, &stringFile{name: absName, text: text})
		}
		for _, name := range badNames {
			delete(r.files, name)
		}
		r.listCache = cache
	})

	switch name {
	case "/":
		return &stringFile{name: name, children: r.listCache}, nil
	default:
		text, ok := r.files[name]
		if !ok {
			return nil, os.ErrNotExist
		}
		return &stringFile{name: name, text: text}, nil
	}
}

type stringFile struct {
	name string
	text string

	children []*stringFile
	buf      *bytes.Buffer
}

func (s *stringFile) Read(p []byte) (n int, err error) {
	if s.buf == nil {
		s.buf = bytes.NewBufferString(s.text)
	}
	return s.buf.Read(p)
}

func (s *stringFile) Close() error {
	return nil
}

func (s *stringFile) Seek(offset int64, whence int) (int64, error) {
	return 0, http.ErrNotSupported
}

func (s *stringFile) Readdir(count int) ([]os.FileInfo, error) {
	if len(s.children) == 0 {
		return nil, nil
	}
	limit := len(s.children)
	if count > 0 && limit > count {
		limit = count
	}
	ret := make([]os.FileInfo, limit)
	for i := 0; i < limit; i++ {
		ret[i] = s.children[i]
	}
	return ret, nil
}

func (s *stringFile) Stat() (os.FileInfo, error) {
	return s, nil
}

// os.FileInfo members
func (s *stringFile) Name() string       { return s.name }
func (s *stringFile) Size() int64        { return int64(len(s.text)) }
func (s *stringFile) Mode() os.FileMode  { return 0644 }
func (s *stringFile) ModTime() time.Time { return time.Time{} }
func (s *stringFile) IsDir() bool        { return s.children != nil }
func (s *stringFile) Sys() interface{}   { return nil }
