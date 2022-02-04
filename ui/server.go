package ui

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/discoverkl/gots/one"
	"golang.org/x/net/websocket"
)

var dev = one.InDevMode("gots")

// ReadyFuncName is an async ready function in api object.
const ReadyFuncName = "Gots"
const ContextBindingName = "context"

type HTTPServer interface {
	Handle(pattern string, handler http.Handler)
}

type HTTPServerFunc func(pattern string, handler http.Handler)

func (f HTTPServerFunc) Handle(pattern string, handler http.Handler) {
	f(pattern, handler)
}

type UIContext struct {
	Request *http.Request
	Done    <-chan bool
}

type ObjectFactory func(*UIContext) interface{}

var defaultServerPath = "/gots"

type ClientOptions struct {
	BlurOnClose bool
}

type FileServer struct {
	Addr          string
	ServerPath    string
	Listener      net.Listener
	Prefix        string // path prefix
	Auth          func(http.HandlerFunc) http.HandlerFunc
	HistoryMode   bool
	ClientOptions *ClientOptions

	root http.FileSystem // optional for default instance

	server   *http.Server
	serveMux *http.ServeMux
	es       []muxEntry

	bindingNames map[string]bool // for js placeholder
	bindings     []Bindings

	// local server done
	wg                   sync.WaitGroup
	once                 sync.Once
	started              chan struct{}
	localServerDone      chan struct{}
	localServerExitDelay time.Duration // -1: never exit 0: no delay
	doneOnce             sync.Once
}
type muxEntry struct {
	h       http.Handler
	pattern string
}

func NewFileServer(root http.FileSystem) *FileServer {
	serveMux := http.NewServeMux()
	s := &FileServer{
		root:                 root,
		serveMux:             serveMux,
		es:                   []muxEntry{},
		server:               &http.Server{Handler: serveMux},
		bindingNames:         map[string]bool{},
		bindings:             []Bindings{},
		started:              make(chan struct{}),
		localServerDone:      make(chan struct{}),
		localServerExitDelay: time.Millisecond * 200,
	}
	go func() {
		<-s.started
		if dev {
			log.Println("server active")
		}
		s.wg.Wait()
		if dev {
			log.Println("server done")
		}
		if s.localServerExitDelay > 0 {
			// log.Printf("delay %v and local done after client lost", s.localServerExitDelay)
			s.closeLocalServer()
		}
	}()
	return s
}

func (s *FileServer) ServeExistingServer(server HTTPServer) {
	s.installHandlers(server, false)
}

func (s *FileServer) ServeExistingServerTLS(server HTTPServer) {
	s.installHandlers(server, true)
}

func (s *FileServer) ListenAndServe() error {
	s.installHandlers(nil, false)
	if s.Listener != nil {
		return s.server.Serve(s.Listener)
	}
	return s.server.ListenAndServe()
}

func (s *FileServer) ListenAndServeTLS(certFile, keyFile string) error {
	s.installHandlers(nil, true)
	if s.Listener != nil {
		return s.server.ServeTLS(s.Listener, certFile, keyFile)
	}
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

func (s *FileServer) handlePage(path string, root http.FileSystem) {
	path = strings.Join([]string{s.getPrefix(), path}, "/")
	path = strings.TrimRight(path, "/")
	// s.serveMux.Handle(path+"/", http.StripPrefix(path, http.FileServer(root)))
	handler := http.FileServer(root)
	if s.HistoryMode {
		handler = s.historyModeFileServer(root)
	}
	s.es = append(s.es, muxEntry{pattern: path + "/", h: http.StripPrefix(path, handler)})
}

func (s *FileServer) historyModeFileServer(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		name := path.Clean(upath)
		f, err := root.Open(name)
		if err != nil {
			if os.IsNotExist(err) {
				// fallback to root
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func (s *FileServer) getPrefix() string {
	prefix := s.Prefix
	if prefix != "" && prefix[0] != '/' {
		panic(fmt.Sprintf("Prefix must start with '/', got: %s", prefix))
	}
	return strings.TrimRight(prefix, "/")
}

func (s *FileServer) installHandlers(realServer HTTPServer, tls bool) {
	prefix := s.getPrefix()
	if dev {
		log.Printf("with prefix: %s", prefix)
	}
	s.handleGots(prefix, tls)
	s.handlePage("", s.root)

	attachMode := (realServer != nil)
	if realServer == nil {
		realServer = s.serveMux
	}

	for _, e := range s.es {
		realServer.Handle(e.pattern, e.h)
	}

	// ignore Addr and Auth for attacth mode
	if attachMode {
		return
	}

	addr := s.Addr
	if addr == "" {
		addr = ":80"
	}
	s.server.Addr = addr

	if s.Auth != nil {
		s.server.Handler = s.Auth(s.serveMux.ServeHTTP)
	}
}

func (s *FileServer) Shutdown(ctx context.Context) error {
	s.closeLocalServer()
	return s.server.Shutdown(context.Background())
}

func (s *FileServer) Close() error {
	s.closeLocalServer()
	return s.server.Close()
}

func (s *FileServer) closeLocalServer() {
	s.doneOnce.Do(func() {
		close(s.localServerDone)
	})
}

func (s *FileServer) handleGots(prefix string, tls bool) {
	serverPath := s.ServerPath
	if serverPath == "" {
		serverPath = defaultServerPath
	}
	if serverPath[0] != '/' {
		panic("serverPath must start with '/'")
	}

	// s.serveMux.Handle(prefix+serverPath, http.StripPrefix(prefix, websocket.Handler(s.serveClientConn)))
	// s.serveMux.Handle(prefix+getScriptPath(serverPath), http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	s.es = append(s.es, muxEntry{pattern: prefix + serverPath, h: http.StripPrefix(prefix, websocket.Handler(s.serveClientConn))})
	s.es = append(s.es, muxEntry{pattern: prefix + getScriptPath(serverPath), h: http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/javascript")
		jsQuery := fmt.Sprintf("?%s", req.URL.RawQuery)

		names := []string{}
		for name := range s.bindingNames {
			names = append(names, name)
		}

		jso := &jsOption{
			TLS:      tls,
			Prefix:   prefix,
			Search:   jsQuery,
			Bindings: names,
		}
		if s.ClientOptions != nil {
			co := s.ClientOptions
			jso.BlurOnClose = co.BlurOnClose
		}
		clientScript := injectOptions(jso)
		fmt.Fprint(w, clientScript)
	}))})
}

func (s *FileServer) Done() <-chan struct{} {
	return s.localServerDone
}

func (s *FileServer) Bind(b Bindings) error {
	if b.Error() != nil {
		return b.Error()
	}
	if err := s.collectBindNames(b); err != nil {
		return err
	}
	s.bindings = append(s.bindings, b)
	return nil
}

func (s *FileServer) collectBindNames(b Bindings) error {
	for _, name := range b.Names() {
		s.bindingNames[name] = true
	}
	return nil
}

// ready(0) -> started(1+) -> done(0)
func (s *FileServer) serveClientConn(ws *websocket.Conn) {
	s.wg.Add(1)
	done := make(chan bool)
	defer func() {
		close(done)
	}()
	defer func() {
		if s.localServerExitDelay > 0 {
			<-time.After(s.localServerExitDelay) // support fast page refresh
		}
		s.wg.Done()
		if s.localServerExitDelay == 0 {
			// log.Printf("local done after client lost")
			s.closeLocalServer()
		}
	}()

	s.once.Do(func() {
		close(s.started)
	})

	p, err := newPage(ws)
	if err != nil {
		log.Printf("attach websocket failed: %v", err)
	}

	// apply binding
	binds := map[string]BindingFunc{}
	collect := func(objName string, target interface{}) {
		objBinds, err := getBindings(objName, target)
		if err != nil {
			log.Printf("get session bindings failed: %v", err)
			return
		}
		for name, f := range objBinds {
			binds[name] = f
		}
	}

	c := &UIContext{Request: ws.Request(), Done: done}
	for _, b := range s.bindings {
		for name, target := range b.Map(c) {
			collect(name, target)
		}
	}

	err = p.bindMap(binds)
	if err != nil {
		log.Printf("binding failed: %v", err)
	}

	// server ready
	err = p.SetReady()
	if err != nil {
		log.Printf("failed to make page ready: %v", err)
	}

	// wait
	<-p.Done()
}

func getScriptPath(serverPath string) string {
	return fmt.Sprintf("%s.js", serverPath)
}

func BasicAuth(auth func(user string, pass string) bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || !auth(user, pass) {
				w.Header().Set("www-authenticate", `Basic realm="nothing"`)
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, http.StatusText(http.StatusUnauthorized))
				return
			}
			handler(w, r)
		}
	}
}
