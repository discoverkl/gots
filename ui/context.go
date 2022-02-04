package ui

import (
	"context"
	"time"
)

type Context struct {
	Seq int `json:"seq"`
	ctx context.Context
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Context) Err() error {
	return c.ctx.Err()
}

func (c *Context) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

func (c *Context) WithCancel() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	return cancel
}
