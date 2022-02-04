package ui

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func sum(a, b int) int {
	return a + b
}

type internalAPI struct {
}

func (*internalAPI) sum(a, b int) int {
	return a + b
}

type someAPI struct {
}

func (*someAPI) Sum(a, b int) int {
	return a + b
}

type someValue struct {
}

func (someValue) Value(a, b int) int {
	return a + b
}

func TestCallable(t *testing.T) {
	assertValid(t, Func("sum", sum))
	assertValid(t, Func("sum", (*internalAPI).sum))
	assertValid(t, Func("sum", (&internalAPI{}).sum))
	assertNotValid(t, Func("sum", &internalAPI{}))
}

func TestObject(t *testing.T) {
	assertNotValid(t, Object(sum))
	assertNotValid(t, Object((*internalAPI).sum))
	assertNotValid(t, Object((&internalAPI{}).sum))
	assertNotValid(t, Object(&internalAPI{}))
	assertValid(t, Object(&someAPI{}))
	assertValid(t, Object(someValue{}))
}

func TestMap(t *testing.T) {
	assertValid(t, Map(map[string]interface{}{"sum": sum}))
}

func TestDelay(t *testing.T) {
	assertValid(t, Delay([]string{"sum"}, func(*UIContext) Bindings {
		return Func("sum", sum)
	}))
	assertValid(t, Delay([]string{"sum"}, func(*UIContext) Bindings {
		return Object(&someAPI{})
	}))
	assertValid(t, Delay([]string{"value"}, func(*UIContext) Bindings {
		return Object(someValue{})
	}))
	assertValid(t, Delay([]string{"value"}, func(c *UIContext) Bindings {
		if c == nil {
			t.Error("empty context")
		}
		return Delay([]string{"value"}, func(c *UIContext) Bindings {
			if c != nil {
				t.Error("non-empty context")
			}
			return Object(someValue{})
		})
	}))
}

func TestDelayObject(t *testing.T) {
	assertValid(t, DelayObject(&someAPI{}, func(*UIContext) Bindings {
		return Object(&someAPI{})
	}))
	assertValid(t, DelayObject(someValue{}, func(*UIContext) Bindings {
		return Object(someValue{})
	}))
	assertNotValid(t, DelayObject(someValue{}, func(*UIContext) Bindings {
		return Object(&someAPI{})
	}))
	assertNotValid(t, DelayObject(&someAPI{}, func(*UIContext) Bindings {
		return Object(someValue{})
	}))
}

func assertValid(t *testing.T, binds Bindings) {
	if err := check(binds); err != nil {
		t.Error(err)
	}
}

func assertNotValid(t *testing.T, binds Bindings) {
	if err := check(binds); err == nil {
		t.Error("should not be valid")
	}
}

func check(binds Bindings) error {
	c := &UIContext{}
	if binds == nil {
		return fmt.Errorf("nil binds")
	}
	if binds.Error() != nil {
		return binds.Error()
	}
	if len(binds.Names()) == 0 {
		return fmt.Errorf("empty binding names")
	}
	if len(binds.Map(c)) == 0 {
		return fmt.Errorf("empty binding map")
	}

	realNames := []string{}
	for name, fn := range binds.Map(c) {
		v := reflect.ValueOf(fn)
		if v.Kind() != reflect.Func {
			return fmt.Errorf("invalid binding target: %v", fn)
		}
		realNames = append(realNames, name)
	}

	names := binds.Names()
	sort.Strings(names)
	sort.Strings(realNames)

	if len(names) != len(realNames) {
		return fmt.Errorf("names and binds are not match")
	}

	for i := 0; i < len(names); i++ {
		if names[i] != realNames[i] {
			return fmt.Errorf("names and binds are not match")
		}
	}
	return nil
}
