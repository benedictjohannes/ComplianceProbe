package elevation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/benedictjohannes/crobe/executor"
)

func RunWorker(socketURI string) error {
	var conn net.Conn
	var err error

	// Retry dialing to allow coordinator listener time to bind/listen
	for i := 0; i < 20; i++ {
		conn, err = Dial(socketURI)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("worker failed to connect to coordinator at %s: %w", socketURI, err)
	}
	defer conn.Close()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var encMu sync.Mutex
	sendResp := func(resp WireMessage) error {
		encMu.Lock()
		defer encMu.Unlock()
		return encoder.Encode(resp)
	}

	type msgResult struct {
		msg WireMessage
		err error
	}

	msgCh := make(chan msgResult, 1)
	go func() {
		for {
			var msg WireMessage
			decodeErr := decoder.Decode(&msg)
			msgCh <- msgResult{msg: msg, err: decodeErr}
			if decodeErr != nil {
				return
			}
		}
	}()

	var activeCancel context.CancelFunc

	for {
		select {
		case <-workerCtx.Done():
			if activeCancel != nil {
				activeCancel()
			}
			return nil
		case res := <-msgCh:
			if res.err != nil {
				workerCancel()
				if activeCancel != nil {
					activeCancel()
				}
				if errors.Is(res.err, io.EOF) || errors.Is(res.err, net.ErrClosed) {
					return nil
				}
				return fmt.Errorf("worker decode error: %w", res.err)
			}

			msg := res.msg
			switch msg.Type {
			case MsgTypeExec:
				if msg.Exec == nil {
					resp := WireMessage{
						Type:  MsgTypeResult,
						ID:    msg.ID,
						Error: "exec request payload missing",
					}
					if err := sendResp(resp); err != nil {
						return err
					}
					continue
				}

				execCtx, cancel := context.WithCancel(workerCtx)
				activeCancel = cancel

				go func(reqID string, execReq executor.ExecutionRequest) {
					defer cancel()
					res, execErr := executor.LocalExecutionRunner.Run(execCtx, execReq)
					resp := WireMessage{
						Type:   MsgTypeResult,
						ID:     reqID,
						Result: &res,
					}
					if execErr != nil {
						resp.Error = execErr.Error()
					}
					_ = sendResp(resp)
				}(msg.ID, *msg.Exec)

			case MsgTypePing:
				resp := WireMessage{
					Type: MsgTypePong,
					ID:   msg.ID,
				}
				if err := sendResp(resp); err != nil {
					return err
				}

			case MsgTypeShutdown:
				workerCancel()
				if activeCancel != nil {
					activeCancel()
				}
				return nil
			}
		}
	}
}
