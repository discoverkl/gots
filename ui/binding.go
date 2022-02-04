package ui

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

type BindingFunc interface{}

// Bindings represent an api.
// Every binding has a name and a callable object.
type Bindings interface {
	Names() []string
	Map(*UIContext) map[string]BindingFunc
	Error() error
}

func Prefix(name string, b Bindings) Bindings {
	if name == "" {
		return &mapBinding{err: fmt.Errorf("bind prefix: name is empty")}
	}
	return &prefixBinding{prefix: name, Bindings: b}
}

func Func(name string, fn interface{}) Bindings {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return &mapBinding{err: fmt.Errorf("bind func %s: value is not callable: %T", name, fn)}
	}
	return bindingItem(name, fn)
}

func Object(obj interface{}) Bindings {
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return &mapBinding{err: fmt.Errorf("bind object: value is not a struct: %T", obj)}
	}
	binds, err := getBindings("", obj)
	return &mapBinding{err: err, binds: binds}
}

func Map(m map[string]interface{}) Bindings {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return &mapBinding{err: fmt.Errorf("bind map: value is not a map: %T", m)}
	}
	binds, err := getBindings("", m)
	return &mapBinding{err: err, binds: binds}
}

func Delay(names []string, factory func(*UIContext) Bindings) Bindings {
	if len(names) == 0 {
		return &mapBinding{err: fmt.Errorf("bind delay: names is empty")}
	}
	return &mapBinding{names: names, factory: factory}
}

func DelayMap(prototype map[string]interface{}, factory func(*UIContext) Bindings) Bindings {
	binds := Map(prototype)
	if binds.Error() != nil {
		return binds
	}
	names := binds.Names()
	if len(names) == 0 {
		return &mapBinding{err: fmt.Errorf("bind delay map: names is empty")}
	}
	return &mapBinding{names: names, factory: factory}
}

func DelayObject(prototype interface{}, factory func(*UIContext) Bindings) Bindings {
	binds := Object(prototype)
	if binds.Error() != nil {
		return binds
	}
	names := binds.Names()
	if len(names) == 0 {
		return &mapBinding{err: fmt.Errorf("bind delay object: names is empty")}
	}
	return &mapBinding{names: names, factory: factory}
}

//
// implements
//

type mapBinding struct {
	names   []string
	binds   map[string]BindingFunc
	factory func(*UIContext) Bindings
	err     error
}

func bindingItem(name string, fn interface{}) *mapBinding {
	return &mapBinding{names: []string{name}, binds: map[string]BindingFunc{name: fn}}
}

// names -> binds -> factory
func (m *mapBinding) Names() []string {
	if m.names != nil {
		return m.names
	}
	if m.binds != nil {
		ret := []string{}
		for name, _ := range m.binds {
			ret = append(ret, name)
		}
		return ret
	}
	return nil
}

// binds -> factory
func (m *mapBinding) Map(c *UIContext) map[string]BindingFunc {
	if m.binds != nil {
		return m.binds
	}
	if m.factory == nil {
		return nil
	}
	return m.factory(c).Map(nil)
}

func (m *mapBinding) Error() error {
	return m.err
}

type member struct {
	Name  string
	Value reflect.Value
}

func getBindings(name string, i interface{}) (map[string]BindingFunc, error) {
	if i == nil {
		return nil, fmt.Errorf("getBindings on nil")
	}
	ret := map[string]BindingFunc{}
	raw := reflect.ValueOf(i)
	v := raw
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Func:
		ret[name] = v.Interface()
		return ret, nil
	case reflect.Map:
		vmap, ok := v.Interface().(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("binding target map is not type of map[string]interface{}")
		}
		for subName, target := range vmap {
			if reflect.TypeOf(target).Kind() != reflect.Func {
				return nil, fmt.Errorf("binding target map value is not callable")
			}
			if name == "" {
				ret[subName] = target
			} else {
				ret[fmt.Sprintf("%s.%s", name, subName)] = target
			}
		}
		return ret, nil
	case reflect.Struct:
		members := []member{}
		for i := 0; i < v.Type().NumField(); i++ {
			members = append(members, member{
				Name:  v.Type().Field(i).Name,
				Value: v.Field(i),
			})
		}

		for i := 0; i < raw.Type().NumMethod(); i++ {
			members = append(members, member{
				Name:  raw.Type().Method(i).Name,
				Value: raw.Method(i),
			})
		}

		for _, f := range members {
			if !unicode.IsUpper(rune(f.Name[0])) {
				continue
			}
			// convert to js binding name
			fname := fmt.Sprintf("%s%s", strings.ToLower(f.Name[0:1]), f.Name[1:])
			if name != "" {
				fname = fmt.Sprintf("%s.%s", name, fname)
			}

			// wrap none-Func fields and bind methods
			fv := f.Value
			var bindingFunc interface{}
			switch fv.Kind() {
			case reflect.Func:
				bindingFunc = fv.Interface()
			default:
				bindingFunc = func() interface{} { return fv.Interface() }
			}
			ret[fname] = bindingFunc
		}
	default:
		return nil, fmt.Errorf("unsupport object kind: %v", v.Kind())
	}
	return ret, nil
}

type prefixBinding struct {
	prefix string
	Bindings
}

func (p *prefixBinding) Names() []string {
	names := p.Bindings.Names()
	ret := make([]string, len(names))
	for i := 0; i < len(names); i++ {
		ret[i] = fmt.Sprintf("%s.%s", p.prefix, names[i])
	}
	return ret
}

func (p *prefixBinding) Map(c *UIContext) map[string]BindingFunc {
	binds := p.Bindings.Map(c)
	ret := map[string]BindingFunc{}
	for name, fn := range binds {
		ret[fmt.Sprintf("%s.%s", p.prefix, name)] = fn
	}
	return ret
}
