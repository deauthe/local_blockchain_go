package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
)

type TCPPeer struct {
	conn     net.Conn
	Outgoing bool
	closed   bool
	mu       sync.RWMutex
}

func (p *TCPPeer) Send(b []byte) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	p.mu.RUnlock()

	_, err := p.conn.Write(b)
	if err != nil {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		return fmt.Errorf("failed to send data: %v", err)
	}
	return nil
}

func (p *TCPPeer) readLoop(rpcCh chan RPC) {
	buf := make([]byte, 4096)
	for {
		n, err := p.conn.Read(buf)
		if err == io.EOF {
			p.mu.Lock()
			p.closed = true
			p.mu.Unlock()
			return
		}
		if err != nil {
			p.mu.Lock()
			p.closed = true
			p.mu.Unlock()
			fmt.Printf("read error: %s", err)
			return
		}

		msg := buf[:n]
		rpcCh <- RPC{
			From:    p.conn.RemoteAddr(),
			Payload: bytes.NewReader(msg),
		}
	}
}

func (p *TCPPeer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return p.conn.Close()
}

type TCPTransport struct {
	peerCh     chan *TCPPeer
	listenAddr string
	listener   net.Listener
}

func NewTCPTransport(addr string, peerCh chan *TCPPeer) *TCPTransport {
	return &TCPTransport{
		peerCh:     peerCh,
		listenAddr: addr,
	}
}

func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	t.listener = ln

	go t.acceptLoop()

	return nil
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("accept error from %+v\n", conn)
			continue
		}

		peer := &TCPPeer{
			conn: conn,
		}

		t.peerCh <- peer
	}
}
