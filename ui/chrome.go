package ui

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/discoverkl/gots/homedir"
)

var ChromeBinary string

type chromeConfig struct {
	x          int
	y          int
	width      int
	height     int
	chromeArgs []string
}

type chromePage struct {
	Window
	cmd        *exec.Cmd
	chromeDone chan struct{}
	conf       chromeConfig
}

// func NewChromePage(root http.FileSystem) Window {
// 	return NewChromeApp(root, 0, 0, -1, -1)
// }

func NewChromeApp(root http.FileSystem, x, y int, width, height int, chromeArgs ...string) Window {
	conf := chromeConfig{x, y, width, height, chromeArgs}
	c := &chromePage{
		cmd:        nil,
		chromeDone: make(chan struct{}),
		conf:       conf,
	}
	c.Window = NewPage(root, c.OpenChrome)
	return c
}

func NewChromeAppMapURL(root http.FileSystem, x, y int, width, height int, mapURL func(net.Listener) string, chromeArgs ...string) Window {
	conf := chromeConfig{x, y, width, height, chromeArgs}
	c := &chromePage{
		cmd:        nil,
		chromeDone: make(chan struct{}),
		conf:       conf,
	}
	c.Window = NewPageMapURL(root, c.OpenChrome, mapURL)
	return c
}

func (c *chromePage) OpenChrome(url string) error {
	// ** native window
	var err error
	x, y, width, height := c.conf.x, c.conf.y, c.conf.width, c.conf.height
	var args []string
	pageMode := (width == -1 && height == -1)

	if pageMode {
		args = append(args, url)
		c.cmd, err = newChromeWithArgs(findChrome(), args...)

	} else {
		// use fix data dir for chrome app
		var dir string
		dir, err = homedir.Dir()
		if err != nil {
			return err
		}
		dir = filepath.Join(dir, "gots-chrome")
		if dev {
			log.Printf("user data dir: %s", dir)
		}

		args = append(defaultAppArgs, fmt.Sprintf("--app=%s", url))
		args = append(args, fmt.Sprintf("--user-data-dir=%s", dir))
		args = append(args, fmt.Sprintf("--window-position=%d,%d", x, y))
		args = append(args, fmt.Sprintf("--window-size=%d,%d", width, height))
		args = append(args, c.conf.chromeArgs...)

		c.cmd, err = newChromeWithArgs(findChrome(), args...)
	}

	if err != nil {
		return err
	}

	// subprocess waiter
	go func() {
		err := c.cmd.Wait()
		if dev {
			log.Printf("chrome wait return: %v", err)
		}
		close(c.chromeDone)
	}()

	return err
}

func (c *chromePage) Close() error {
	c.ensureAppClosed()
	return nil
}

func newChromeWithArgs(chromeBinary string, args ...string) (*exec.Cmd, error) {
	if dev {
		log.Println("[chrome-args]:", args)
	}
	if chromeBinary == "" {
		return nil, fmt.Errorf("could not find chrome in your system")
	}

	cmd := exec.Command(chromeBinary, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	log.Println("pid:", cmd.Process.Pid)

	return cmd, nil
}

// for some app only, not page ( chrome subprocess id is not stable )
func (c *chromePage) ensureAppClosed() {
	// close chrome process (for app mode)
	if state := c.cmd.ProcessState; state == nil || !state.Exited() {
		err := c.cmd.Process.Signal(os.Interrupt) // DO NOT kill -> enable gracefully exit
		if err != nil {
			log.Println("kill chrome process error:", err)
		}
	}
	//TODO: timeout and force kill
	<-c.chromeDone
}

//
// tool functions
//

// https://peter.sh/experiments/chromium-command-line-switches/
var defaultAppArgs = []string{
	// "--disable-background-networking",
	// "--disable-background-timer-throttling",
	// "--disable-backgrounding-occluded-windows",
	// "--disable-breakpad",
	// "--disable-client-side-phishing-detection",
	"--disable-default-apps",
	// "--disable-dev-shm-usage",
	"--disable-infobars",
	"--disable-extensions",
	"--disable-features=site-per-process",
	"--disable-hang-monitor",
	"--disable-ipc-flooding-protection",
	"--disable-popup-blocking",
	"--disable-prompt-on-repost",
	// "--disable-renderer-backgrounding",
	"--disable-sync",
	"--disable-translate",
	// "--metrics-recording-only",
	// "--no-first-run",
	"--safebrowsing-disable-auto-update",
	// "--enable-automation",
	// "--password-store=basic",
	// "--use-mock-keychain",

	"--disable-device-discovery-notifications",
}

func findChrome() string {
	if ChromeBinary != "" {
		return ChromeBinary
	}
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
	case "windows":
		paths = []string{
			"C:/Users/" + os.Getenv("USERNAME") + "/AppData/Local/Google/Chrome/Application/chrome.exe",
			"C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
			"C:/Program Files/Google/Chrome/Application/chrome.exe",
			"C:/Users/" + os.Getenv("USERNAME") + "/AppData/Local/Chromium/Application/chrome.exe",
		}
	default:
		paths = []string{
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		return path
	}
	return ""
}
