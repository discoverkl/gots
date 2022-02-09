package ui

import (
	"os"
	"path/filepath"

	"github.com/webview/webview"
)

func OpenWebApp(url string, x, y, width, height int) error {
	w := webview.New(false)
	defer w.Destroy()

	// title
	title, err := os.Executable()
	if err != nil {
		title = ""
	}
	title = filepath.Base(title)
	w.SetTitle(title)

	// size
	w.SetSize(width, height, webview.HintNone)

	// url
	w.Navigate(url)
	w.Run()
	return nil
}
