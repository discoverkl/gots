package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"golang.org/x/net/websocket"
)

// Page of a javascript client.
type Page interface {
	Bind(name string, f interface{}) error
	Eval(js string) Value
	SetReady() error // nofity server ready ( all functions binded )
	Close()
	Done() <-chan struct{}
}

type page struct {
	jsc *jsClient
}

func newPage(ws *websocket.Conn) (*page, error) {
	jsc, err := newJSClient(ws)
	if err != nil {
		return nil, err
	}
	p := &page{
		jsc: jsc,
		// done: make(chan struct{}),
	}
	return p, nil
}

// Bind a Func, map[string]Func or an object
func (c *page) Bind(name string, f interface{}) error {
	binds, err := getBindings(name, f)
	if err != nil {
		return err
	}
	return c.bindMap(binds)
}

func (c *page) bindMap(items map[string]BindingFunc) error {
	for name, f := range items {
		if err := checkBindFunc(name, f); err != nil {
			return err
		}
	}

	binds := map[string]bindingFunc{}
	for name, f := range items {
		v := reflect.ValueOf(f)
		bindingFunc := func(raw []json.RawMessage) (interface{}, error) {
			// Gots.call -> here(do the real call) -> eval for promise
			if len(raw) != v.Type().NumIn() {
				return nil, fmt.Errorf("function arguments mismatch")
			}
			args := []reflect.Value{}

			// TODO: argumets rewrite
			functionType := reflect.TypeOf((**Function)(nil))
			contextType := reflect.TypeOf((*context.Context)(nil))
			for i := range raw {
				// ** process functionType and contxtType
				arg := reflect.New(v.Type().In(i))

				isContext := false
				if arg.Type() == contextType {
					isContext = true
					arg = reflect.New(reflect.TypeOf((*Context)(nil))) // rewrite context.Context interface to ui.Context type
				}

				if err := json.Unmarshal(raw[i], arg.Interface()); err != nil {
					return nil, err
				}

				if isContext {
					ctx := arg.Elem().Interface().(*Context)
					if ctx == nil {
						ctx = &Context{}
						arg.Elem().Set(reflect.ValueOf(ctx))
					}
					cancel := ctx.WithCancel()
					defer cancel()
					c.jsc.ref(ctx.Seq, cancel)
					defer c.jsc.unref(ctx.Seq)

				} else if arg.Type() == functionType {
					fn, _ := arg.Elem().Interface().(*Function)
					if fn != nil {
						fn.jsc = c.jsc
					}
					defer fn.close()
				}
				args = append(args, arg.Elem())
			}

			errorType := reflect.TypeOf((*error)(nil)).Elem()
			res := v.Call(args)
			switch len(res) {
			case 0:
				// no return value
				return nil, nil
			case 1:
				// return value or error
				if res[0].Type().Implements(errorType) {
					if res[0].Interface() != nil {
						return nil, res[0].Interface().(error)
					}
					return nil, nil
				}
				return res[0].Interface(), nil
			case 2:
				// first one is value, second is error
				if !res[1].Type().Implements(errorType) {
					return nil, errors.New("second return value must be an error")
				}
				if res[1].Interface() == nil {
					return res[0].Interface(), nil
				}
				return res[0].Interface(), res[1].Interface().(error)
			default:
				return nil, errors.New("unexpected number of return values")
			}
		}
		binds[name] = bindingFunc
	}
	return c.jsc.bind(binds)
}

func (c *page) Eval(js string) Value {
	v, err := c.jsc.eval(js)
	return value{err: err, raw: v}
}

func (c *page) SetReady() error {
	return c.jsc.ready()
}

func (c *page) Close() {
	c.jsc.cancel()
}

// Done = jsClient done
func (c *page) Done() <-chan struct{} {
	return c.jsc.done
}

func checkBindFunc(name string, f interface{}) error {
	switch name {
	case "":
		return fmt.Errorf("name is required for function binding")
	case ReadyFuncName:
		return fmt.Errorf("binding name '%s' is reserved for internal use", name)
	case ContextBindingName:
		return fmt.Errorf("binding name '%s' is reserved for javascript context package", name)
	}

	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("%s: should be a function", name)
	}
	if n := v.Type().NumOut(); n > 2 {
		return fmt.Errorf("%s: too many return values", name)
	}
	return nil
}
