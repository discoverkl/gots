//go:build !(webview || windows || darwin)

package ui

import (
	"fmt"
)

func OpenWebApp(url string, x, y, width, height int) error {
	return fmt.Errorf("webview is not supported in your platform")
}
