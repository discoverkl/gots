package ui

const (
	modApp    = "app"
	modPage   = "page"
	modOnline = "online"
)

type RunMode interface {
	IsApp() bool
	IsPage() bool
	IsOnline() bool
	IsLocal() bool
}

type runMode string

func (r runMode) IsApp() bool {
	return r == modApp
}

func (r runMode) IsPage() bool {
	return r == modPage
}

func (r runMode) IsOnline() bool {
	return r == modOnline
}

func (r runMode) IsLocal() bool {
	return r.IsApp() || r.IsPage()
}

func (r runMode) Empty() bool {
	return r == ""
}
