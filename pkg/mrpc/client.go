package mrpc

import (
	"context"
	"errors"
	"net"
	"net/rpc"
	"sync"
	"task/pkg/proto"
	"time"
)

const (
	diaTimeout   = 5 * time.Second
	callTimeout  = 1 * time.Minute
	pingDuration = 10 * time.Second
	pingMethod   = "Serve.Ping"
)

var (
	rpcClients    *clients
	errRpc        = errors.New("mrpc is not available")
	errRpcTimeout = errors.New("mrpc call timeout")
	errRpcCancel  = errors.New("mrpc call cancel")
)

func init() {
	rpcClients = &clients{
		clients: make(map[string]*client),
	}
}

func Call(addr string, method string, ctx context.Context, args interface{}, reply interface{}) error {
	c := rpcClients.get(addr)
	if c == nil {
		return errRpc
	}
	err := c.call(method, ctx, args, reply)
	if errors.Is(err, rpc.ErrShutdown) {
		rpcClients.del(addr)
	}
	return err
}

type clients struct {
	clients map[string]*client
	mux     sync.RWMutex
}

func (cs *clients) get(addr string) *client {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if c, ok := cs.clients[addr]; ok {
		return c
	}
	c := newClient(options{
		network: "tcp4",
		addr:    addr,
	})
	if c.err != nil {
		return nil
	}
	cs.clients[addr] = c
	go c.ping()
	return c
}

func (cs *clients) del(addr string) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if c, ok := cs.clients[addr]; ok {
		c.close()
	}
	delete(cs.clients, addr)
}

type options struct {
	network string
	addr    string
}

type client struct {
	client  *rpc.Client
	options options
	quit    chan struct{}
	err     error
}

func newClient(options options) *client {
	c := &client{}
	c.options = options
	c.quit = make(chan struct{}, 100)
	c.err = c.dial()
	return c
}

func (c *client) dial() (err error) {
	conn, err := net.DialTimeout(c.options.network, c.options.addr, diaTimeout)
	if err != nil {
		return err
	}
	c.client = rpc.NewClient(conn)
	return nil
}

func (c *client) call(method string, ctx context.Context, args interface{}, reply interface{}) error {
	select {
	case <-ctx.Done():
		return errRpcCancel
	case call := <-c.client.Go(method, args, reply, make(chan *rpc.Call, 1)).Done:
		return call.Error
	case <-time.After(callTimeout):
		return errRpcTimeout
	}
}

func (c *client) close() {
	c.quit <- struct{}{}
}

func (c *client) ping() {
	var err error
	for {
		select {
		case <-c.quit:
			if c.client != nil {
				_ = c.client.Close()
				break
			}
		default:
			if c.client != nil && c.err == nil {
				if err = c.call(pingMethod, context.TODO(), &proto.EmptyArgs{}, &proto.EmptyReply{}); err != nil {
					c.err = err
					_ = c.client.Close()
				}
			} else {
				if err = c.dial(); err == nil {
					c.err = nil
				}
			}
			time.Sleep(pingDuration)
		}
	}
}
