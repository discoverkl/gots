package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	// "github.com/google/shlex"
)

type UI interface {
	Run() error
	Bindable
	RunMode
	Add(name string, child UI) // add sub UI
}

type Bindable interface {
	Bind(b Bindings)
	BindPrefix(name string, b Bindings)
	BindFunc(name string, fn interface{})
	BindObject(obj interface{})
	BindMap(m map[string]interface{})
	GetBindings() []Bindings
}
type Binder interface {
	Bind(b Bindings) error
}

type ui struct {
	conf *uiConfig
	runMode
	confError error
	bindings  []Bindings
	children  map[string]UI
}

func New(ops ...Option) UI {
	var err error
	var confError error

	conf := defaultUIConfig()
	for _, op := range ops {
		err = op(conf)
		if err != nil && confError == nil {
			confError = fmt.Errorf("ui config: %w", err)
		}
	}

	app := &ui{conf: conf, confError: confError, children: map[string]UI{}}
	app.useRunMode()
	app.useSpecialEnvSetting()
	return app
}

func (u *ui) Bind(b Bindings) {
	u.bindings = append(u.bindings, b)
}

func (u *ui) BindPrefix(name string, b Bindings) {
	u.Bind(Prefix(name, b))
}

func (u *ui) BindFunc(name string, fn interface{}) {
	u.Bind(Func(name, fn))
}

func (u *ui) BindObject(obj interface{}) {
	u.Bind(Object(obj))
}

func (u *ui) BindMap(m map[string]interface{}) {
	u.Bind(Map(m))
}

func (u *ui) GetBindings() []Bindings {
	ret := make([]Bindings, len(u.bindings))
	for i := 0; i < len(u.bindings); i++ {
		ret[i] = u.bindings[i]
	}
	return ret
}

func (u *ui) Run() error {
	c := u.conf

	if u.confError != nil {
		return u.confError
	}

	if !c.Quiet {
		log.Println("run mode:", u.runMode)
	}
	if u.runMode.Empty() {
		return fmt.Errorf("run mode is not set")
	}

	var win Window
	var svr *FileServer
	var err error

	// ** create window or server
	switch true {
	case u.IsApp():
		if c.AppChromeBinary != "" {
			ChromeBinary = c.AppChromeBinary
		}
		if c.LocalMapURL == nil {
			win = NewApp(c.Root, c.AppX, c.AppY, c.AppWidth, c.AppHeight, c.AppChromeArgs...)
		} else {
			win = NewAppMapURL(c.Root, c.AppX, c.AppY, c.AppWidth, c.AppHeight, c.LocalMapURL, c.AppChromeArgs...)
		}
		svr = win.Server()
	case u.IsPage():
		if c.LocalMapURL == nil {
			win = NewPage(c.Root)
		} else {
			win = NewPageMapURL(c.Root, c.LocalMapURL)
		}
		svr = win.Server()
	case u.IsOnline():
		svr = NewFileServer(c.Root)
		svr.Addr = c.OnlineAddr
		svr.Listener = c.OnlineListener
		svr.Prefix = c.OnlinePrefix
		svr.Auth = c.OnlineAuth
	default:
		return fmt.Errorf("unsupported mode: %v", u)
	}

	// ** Client Options
	svr.HistoryMode = u.conf.HistoryMode
	svr.ClientOptions = &ClientOptions{
		BlurOnClose: u.conf.BlurOnClose,
	}

	// ** Bindings
	for _, b := range u.bindings {
		err = svr.Bind(b)
		if err != nil {
			return err
		}
	}
	for name, childUI := range u.children {
		child, ok := childUI.(*ui)
		if !ok {
			continue
		}
		svr.handlePage(name, child.conf.Root)
		for _, b := range child.bindings {
			err = svr.Bind(b)
			if err != nil {
				return err
			}
		}
	}

	// ** Run
	switch true {
	case u.IsLocal():
		if c.LocalExitDelay != nil {
			win.SetExitDelay(*c.LocalExitDelay)
		}
		return win.Open()
	case u.IsOnline():
		if c.OnlineAttach != nil {
			if c.OnlineAttachTLS {
				svr.ServeExistingServerTLS(c.OnlineAttach)
			} else {
				svr.ServeExistingServer(c.OnlineAttach)
			}
			return nil
		}
		if !c.Quiet && svr.Addr != "" {
			log.Printf("listen on: %s", svr.Addr)
		}
		if c.OnlineCertFile != "" && c.OnlineKeyFile != "" {
			return svr.ListenAndServeTLS(c.OnlineCertFile, c.OnlineKeyFile)
		}
		return svr.ListenAndServe()
	default:
		panic("not here")
	}
}

func (u *ui) Add(name string, child UI) {
	u.children[name] = child
}

func (u *ui) Done() <-chan struct{} {
	panic(nil)
}

//
// private methods
//

func (u *ui) useRunMode() {
	// get mode from env
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = os.Getenv("mode")
	}
	mode = strings.ToLower(mode)
	override := strings.HasSuffix(mode, "!")
	mode = strings.TrimRight(mode, "!")

	if !override && u.conf.Mode != "" {
		mode = u.conf.Mode
	}

	switch mode {
	case "app":
	case "page":
	case "online":
	default:
		if u.conf.OnlineAttach != nil {
			mode = "online"
		} else {
			mode = "page"
		}
	}
	u.runMode = runMode(mode)
}

func (u *ui) useSpecialEnvSetting() {
	// online port
	portEnv := os.Getenv("ONLINE_PORT")
	if portEnv != "" {
		override := strings.HasSuffix(portEnv, "!")
		portEnv = strings.TrimRight(portEnv, "!")
		if u.conf.OnlineAddr == "" || override {
			u.conf.OnlineAddr = fmt.Sprintf(":%s", portEnv)
		}
	}

	// // chrome args
	// chromeEnv := os.Getenv("APP_CHROME_ARGS")
	// if chromeEnv != "" {
	// 	chromeArgs, err := shlex.Split(chromeEnv)
	// 	if err != nil {
	// 		log.Printf("parse env arguments failed for APP_CHROME_ARGS: %v", err)
	// 	} else {
	// 		u.conf.AppChromeArgs = append(u.conf.AppChromeArgs, chromeArgs...)
	// 	}
	// }

	// chrome binary
	chromePathEnv := os.Getenv("APP_CHROME_BINARY")
	if chromePathEnv != "" {
		u.conf.AppChromeBinary = chromePathEnv
	}
}
