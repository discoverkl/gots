package ui

import "testing"

func TestIsApp(t *testing.T) {
	mode := runMode("app")
	if !mode.IsApp() {
		t.Error("mode.IsApp() = false")
	}
	if mode.IsOnline() {
		t.Error("mode.IsOnline() = true")
	}
	if runMode("xxx").IsApp() {
		t.Error()
	}
	if runMode("APP").IsApp() {
		t.Error()
	}
	if runMode("App").IsApp() {
		t.Error()
	}
	if runMode("").IsApp() {
		t.Error()
	}
}

func TestIsLocal(t *testing.T) {
	if runMode("online").IsLocal() {
		t.Error()
	}
	if runMode("local").IsLocal() {
		t.Error()
	}
	if !runMode("app").IsLocal() {
		t.Error()
	}
	if !runMode("page").IsLocal() {
		t.Error()
	}
	if runMode("xxx").IsLocal() {
		t.Error()
	}
}
