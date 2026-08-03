package elevation

import "github.com/benedictjohannes/crobe/executor"

type MessageType string

const (
	MsgTypeExec     MessageType = "exec"
	MsgTypeResult   MessageType = "result"
	MsgTypePing     MessageType = "ping"
	MsgTypePong     MessageType = "pong"
	MsgTypeShutdown MessageType = "shutdown"
)

type WireMessage struct {
	Type   MessageType                `json:"type"`
	ID     string                     `json:"id,omitempty"`
	Exec   *executor.ExecutionRequest `json:"exec,omitempty"`
	Result *executor.ExecutionResult  `json:"result,omitempty"`
	Error  string                     `json:"error,omitempty"`
}
