package transport

import (
	"errors"
	"sync"
)

// ErrClosed is returned by a closed connection.
var ErrClosed = errors.New("transport: connection closed")

// Pipe returns a pair of connected in-process Conns. It models a WebRTC data
// channel for tests and the headless harness without real networking; the
// pion-backed Transport (New) is the production path. Bytes written to one end
// are read from the other.
func Pipe() (Conn, Conn) {
	a2b := make(chan []byte, 64)
	b2a := make(chan []byte, 64)
	done := make(chan struct{})
	var once sync.Once
	closeFn := func() { once.Do(func() { close(done) }) }
	a := &pipeConn{out: a2b, in: b2a, done: done, closeFn: closeFn}
	b := &pipeConn{out: b2a, in: a2b, done: done, closeFn: closeFn}
	return a, b
}

type pipeConn struct {
	out     chan []byte
	in      chan []byte
	done    chan struct{}
	closeFn func()
}

func (c *pipeConn) Send(b []byte) error {
	cp := append([]byte(nil), b...)
	select {
	case <-c.done:
		return ErrClosed
	case c.out <- cp:
		return nil
	}
}

func (c *pipeConn) Recv() ([]byte, error) {
	select {
	case <-c.done:
		// Drain any buffered message before reporting closed.
		select {
		case b := <-c.in:
			return b, nil
		default:
			return nil, ErrClosed
		}
	case b := <-c.in:
		return b, nil
	}
}

func (c *pipeConn) Close() error {
	c.closeFn()
	return nil
}
