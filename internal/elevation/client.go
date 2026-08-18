package elevation

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benedictjohannes/crobe/executor"
)

type Client struct {
	listener  net.Listener
	conn      net.Conn
	encoder   *json.Encoder
	decoder   *json.Decoder
	socketURI string
	proc      *ProcessHandle
	seq       uint64
	mu        sync.Mutex
	closed    bool
}

func NewClient() (*Client, error) {
	socketURI := GenerateSocketURI()
	listener, err := Listen(socketURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPC listener: %w", err)
	}

	proc, err := SpawnWorker(socketURI)
	if err != nil {
		listener.Close()
		CleanupListener(socketURI)
		return nil, fmt.Errorf("failed to spawn elevated worker: %w", err)
	}

	type acceptRes struct {
		conn net.Conn
		err  error
	}
	ch := make(chan acceptRes, 1)
	go func() {
		c, e := listener.Accept()
		ch <- acceptRes{conn: c, err: e}
	}()

	var conn net.Conn
	select {
	case res := <-ch:
		if res.err != nil {
			listener.Close()
			CleanupListener(socketURI)
			if proc != nil {
				_ = proc.Kill()
			}
			return nil, fmt.Errorf("failed accepting worker connection: %w", res.err)
		}
		conn = res.conn
	case <-time.After(60 * time.Second):
		listener.Close()
		CleanupListener(socketURI)
		if proc != nil {
			_ = proc.Kill()
		}
		return nil, fmt.Errorf("timeout waiting for elevated worker to connect")
	}

	return &Client{
		listener:  listener,
		conn:      conn,
		encoder:   json.NewEncoder(conn),
		decoder:   json.NewDecoder(conn),
		socketURI: socketURI,
		proc:      proc,
	}, nil
}

func NewClientWithConn(conn net.Conn, listener net.Listener, socketURI string, proc *ProcessHandle) *Client {
	return &Client{
		listener:  listener,
		conn:      conn,
		encoder:   json.NewEncoder(conn),
		decoder:   json.NewDecoder(conn),
		socketURI: socketURI,
		proc:      proc,
	}
}

func (c *Client) Run(ctx context.Context, req executor.ExecutionRequest) (executor.ExecutionResult, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return executor.ExecutionResult{}, fmt.Errorf("client closed")
	}

	select {
	case <-ctx.Done():
		c.mu.Unlock()
		return executor.ExecutionResult{}, ctx.Err()
	default:
	}

	reqID := fmt.Sprintf("req-%d", atomic.AddUint64(&c.seq, 1))
	msg := WireMessage{
		Type: MsgTypeExec,
		ID:   reqID,
		Exec: &req,
	}

	err := c.encoder.Encode(msg)
	c.mu.Unlock()

	if err != nil {
		return executor.ExecutionResult{}, fmt.Errorf("failed to send exec request: %w", err)
	}

	type decodeRes struct {
		resp WireMessage
		err  error
	}
	ch := make(chan decodeRes, 1)
	go func() {
		var resp WireMessage
		err := c.decoder.Decode(&resp)
		ch <- decodeRes{resp: resp, err: err}
	}()

	select {
	case <-ctx.Done():
		return executor.ExecutionResult{}, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return executor.ExecutionResult{}, fmt.Errorf("failed to receive exec response: %w", res.err)
		}

		resp := res.resp
		if resp.ID != reqID {
			return executor.ExecutionResult{}, fmt.Errorf("mismatched response ID: got %s, want %s", resp.ID, reqID)
		}

		if resp.Error != "" {
			resResult := executor.ExecutionResult{}
			if resp.Result != nil {
				resResult = *resp.Result
			}
			return resResult, fmt.Errorf("elevated worker error: %s", resp.Error)
		}

		if resp.Result == nil {
			return executor.ExecutionResult{}, fmt.Errorf("nil result from elevated worker")
		}

		return *resp.Result, nil
	}
}

func (c *Client) Ping() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client closed")
	}

	pingID := fmt.Sprintf("ping-%d", atomic.AddUint64(&c.seq, 1))
	msg := WireMessage{
		Type: MsgTypePing,
		ID:   pingID,
	}

	err := c.encoder.Encode(msg)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	var resp WireMessage
	if err := c.decoder.Decode(&resp); err != nil {
		return fmt.Errorf("failed to receive pong: %w", err)
	}

	if resp.Type != MsgTypePong || resp.ID != pingID {
		return fmt.Errorf("unexpected ping response: type=%s, id=%s", resp.Type, resp.ID)
	}

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true

	_ = c.encoder.Encode(WireMessage{Type: MsgTypeShutdown})
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.listener != nil {
		_ = c.listener.Close()
	}
	CleanupListener(c.socketURI)
	proc := c.proc
	c.mu.Unlock()

	if proc != nil {
		_ = proc.Wait()
	}

	return nil
}
