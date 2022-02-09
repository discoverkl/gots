package ui

import "github.com/webview/webview"

func OpenWebApp(url string, x, y, width, height int) error {
	w := webview.New(false)
	defer w.Destroy()
	w.SetSize(width, height, webview.HintNone)
	w.Navigate(url)
	w.Run()
	return nil
}
