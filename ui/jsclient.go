package ui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"golang.org/x/net/websocket"
)

type result struct {
	Value json.RawMessage
	Err   error
}

type bindingFunc func(args []json.RawMessage) (interface{}, error)

type msg struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	// Result json.RawMessage `json:"result"`
	// Error  json.RawMessage `json:"error"`
}

type retParams struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

type callParams struct {
	Name string            `json:"name"`
	Seq  int               `json:"seq"`
	Args []json.RawMessage `json:"args"`
}

type refCallParams struct {
	Seq int `json:"seq"`
}

type h map[string]interface{}

type jsClient struct {
	sync.Mutex
	id      int32
	pending map[int]chan result
	ws      *websocket.Conn
	binding map[string]bindingFunc
	refs    map[int]func() // int -> func()
	done    chan struct{}  // done = readLoop() return = receive EOF
	cancel  context.CancelFunc
}

func newJSClient(ws *websocket.Conn) (*jsClient, error) {
	p := &jsClient{
		ws:      ws,
		pending: map[int]chan result{},
		binding: map[string]bindingFunc{},
		refs:    map[int]func(){},
		done:    make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.readLoop(ctx)
	return p, nil
}

func (p *jsClient) readLoop(ctx context.Context) {
	defer close(p.done)

	// connection closer
	go func() {
		<-ctx.Done()
		p.ws.Close()
	}()

	for {
		m := msg{}
		if err := websocket.JSON.Receive(p.ws, &m); err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("remote closed")
				return
			}
			if ctx.Err() != nil {
				// cancel
				return
			}
			log.Println("receive bad message:", err)
			p.ws.Close()
			break
			// continue
		}
		if dev {
			log.Printf("[receive] %s, param: %v", m.Method, string(m.Params))
		}

		switch m.Method {
		case "Gots.ret":
			ret := retParams{}
			err := json.Unmarshal([]byte(m.Params), &ret)
			if err != nil {
				log.Println("Gots.ret bad message:", err)
				// DO NOT break
			}

			p.Lock()
			retCh, ok := p.pending[m.ID]
			delete(p.pending, m.ID)
			p.Unlock()

			if !ok {
				var v interface{}
				err = json.Unmarshal(ret.Result, &v)
				valid := (err == nil)
				log.Printf("ignore Gots.ret %d: valid=%v ret=%v, err=%s", m.ID, valid, v, ret.Error)
				continue
			}

			if ret.Error != "" {
				retCh <- result{Err: errors.New(ret.Error)}
			} else {
				retCh <- result{Value: ret.Result}
			}
		case "Gots.call":
			call := callParams{}
			err := json.Unmarshal([]byte(m.Params), &call)
			if err != nil {
				log.Println("Gots.call bad message:", err)
				break
			}

			p.Lock()
			binding, ok := p.binding[call.Name]
			p.Unlock()

			if !ok {
				break
			}

			go func() {
				// jsRet is null or string, jsErr is json value
				var jsRet, jsErr interface{}
				// binding call phrase 2
				if ret, err := binding(call.Args); err != nil {
					jsErr = err.Error()
				} else if _, err = json.Marshal(ret); err != nil {
					jsErr = err.Error()
				} else {
					jsRet = ret
				}
				_, err = p.send("Gots.ret", h{"name": call.Name, "seq": call.Seq, "result": jsRet, "error": jsErr}, false)
				if err != nil {
					log.Println("binding call phrase 3 failed:", err)
				}
			}()
		case "Gots.refCall":
			refCall := refCallParams{}
			err := json.Unmarshal([]byte(m.Params), &refCall)
			if err != nil {
				log.Println("Gots.refCall bad message:", err)
				break
			}
			fn, ok := p.refs[refCall.Seq]
			if !ok {
				// log.Println("Gots.refCall ignore late cancel")
				break
			}
			fn()

		default:
			log.Println("unknown method:", m.Method)
		}
	}
}

func (p *jsClient) send(method string, params h, wait bool) (json.RawMessage, error) {
	if dev {
		log.Printf("   [send] method %s, wait=%v", method, wait)
	}
	id := atomic.AddInt32(&p.id, 1)
	m := h{"id": int(id), "method": method, "params": params}

	var retCh chan result
	if wait {
		retCh = make(chan result)
		p.Lock()
		p.pending[int(id)] = retCh
		p.Unlock()
	}

	err := websocket.JSON.Send(p.ws, m)
	if err != nil {
		// TODO: remove item in p.pending
		return nil, err
	}

	if !wait {
		return nil, nil
	}
	ret := <-retCh
	return ret.Value, ret.Err
}

func (p *jsClient) eval(expr string) (json.RawMessage, error) {
	return p.send("Gots.call", h{"name": "eval", "args": []string{expr}}, true)
}

func (p *jsClient) bind(items map[string]bindingFunc) error {
	added := []string{}
	p.Lock()
	for name, f := range items {
		_, exists := p.binding[name]
		p.binding[name] = f
		if !exists {
			added = append(added, name)
		}
	}
	p.Unlock()

	if len(added) == 0 {
		return nil
	}

	if _, err := p.send("Gots.bind", h{"name": added}, false); err != nil {
		return err
	}
	return nil
}

func (p *jsClient) ready() error {
	if _, err := p.send("Gots.ready", nil, false); err != nil {
		return err
	}
	return nil
}

func (p *jsClient) ref(seq int, fn func()) {
	p.refs[seq] = fn
}

func (p *jsClient) unref(seq int) {
	delete(p.refs, seq)
}
