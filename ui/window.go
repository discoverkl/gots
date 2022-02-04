package ui

import "time"

type Window interface {
	Bind(b Bindings) error
	Open() error
	Server() *FileServer
	SetExitDelay(d time.Duration)
	Eval(js string) Value
	Done() <-chan struct{}
	Close() error
}
