package code

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"

	"github.com/discoverkl/gots/ui"
)

//go:embed fe/dist
var root embed.FS

type API struct {
	root http.FileSystem
}

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

func (a *API) ListDir(path string) ([]FileInfo, error) {
	f, err := a.root.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("'%s' is not dir", path)
	}
	fis, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	ret := []FileInfo{}
	for _, fi := range fis {
		ret = append(ret, FileInfo{
			Name:  fi.Name(),
			Path:  filepath.Join(path, fi.Name()),
			IsDir: fi.IsDir(),
		})
	}

	sort.Slice(ret, func(i, j int) bool {
		l, r := ret[i], ret[j]
		if l.IsDir == r.IsDir {
			return l.Name < r.Name
		}
		return l.IsDir
	})

	return ret, nil
}

func (a *API) LoadText(path string) (string, error) {
	f, err := a.root.Open(path)
	if err != nil {
		return "", err
	}
	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (a *API) SaveText(path string, text string) error {
	return fmt.Errorf("not supported")
}

func UI(codeRoot http.FileSystem, ops ...ui.Option) ui.UI {
	www, _ := fs.Sub(root, "fe/dist")
	ops = append([]ui.Option{
		ui.Root(http.FS(www)),
		ui.OnlinePort(8000),
	}, ops...)

	app := ui.New(ops...)
	app.BindObject(&API{root: codeRoot})
	return app
}
