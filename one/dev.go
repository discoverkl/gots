package one

import (
	"fmt"
	"os"
)

func InDevMode(packageName string) bool {
	var value string
	if packageName != "" {
		key := fmt.Sprintf("dev_%s", packageName)
		value = os.Getenv(key)
		if value != "" {
			return value == "1"
		}
	}
	value = os.Getenv("dev")
	return value == "1"
}
