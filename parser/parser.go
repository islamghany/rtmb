package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

const maxMessageSize = 1024 * 1024 // 1 MB

type CmdName string

const (
	PUB     CmdName = "PUB"
	SUB     CmdName = "SUB"
	UNSUB   CmdName = "UNSUB"
	MSG     CmdName = "MSG"
	PONG    CmdName = "PONG"
	PING    CmdName = "PING"
	INFO    CmdName = "INFO"
	CONNECT CmdName = "CONNECT"
)

type Cmd struct {
	Name    CmdName
	Bytes   []byte // the body of the message for PUB
	Subject string // the subject for PUB, SUB, UNSUB e.g. "foo", "foo.bar", ">", etc.
	ID      string // the unique subscription ID that the client assign to the subscription his ID helps the client track which messages belong to which subscription.
	// (optional): If specified, the subscription becomes a part of a queue group. This is used for load balancing messages among multiple clients.
	// All clients that subscribe to the same subject with the same queue group name will form a queue group,
	// and the server will deliver each message to one of the clients in the group.
	QueueGroup             string // Queue group name for load-balanced message delivery
	ConnectData            ConnectCommand
	MaxMessagesBeforeUnsub int // The maximum number of messages to receive before unsubscribing
}

type ConnectCommand struct {
	Name      string `json:"name,omitempty"`
	Protocol  int    `json:"protocol,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"-"`
	Verbose   bool   `json:"verbose,omitempty"`
}

// Parse reads a command from an io.Reader and parses it into a Cmd struct
func Parse(reader io.Reader) (*Cmd, error) {
	buffReader := bufio.NewReader(reader)
	// Read until newline without allocation
	line, err := buffReader.ReadSlice('\n')
	if err != nil {
		return nil, err
	}

	// Find the first space to separate the command name
	spaceIndex := bytes.IndexByte(line, ' ')

	if spaceIndex < 0 {
		spaceIndex = bytes.IndexByte(line, '\n') - 1
		if spaceIndex < 0 {
			return nil, fmt.Errorf("Empty Command")
		}
	}

	cmdName := CmdName(bytes.ToUpper(line[:spaceIndex]))
	cmd := &Cmd{Name: cmdName}

	switch cmdName {
	case PUB: // e.g. PUB foo 5\r\nhello , PUB foo.bar 10 5\r\nhello
		return cmd.parsePUB(line[spaceIndex+1:], buffReader)
	case SUB: // e.g. SUB foo 1, SUB foo  bar.*.baz 2
		return cmd.parseSUB(line[spaceIndex+1:])
	case CONNECT: // e.g. CONNECT {"name":"client","verbose":true} or CONNECT {}
		return cmd.parseCONNECT(line[spaceIndex+1:])
	case PONG: // e.g. PONG\r\n
		return cmd, nil
	case UNSUB: // e.g. UNSUB foo, UNSUB foo 5
		return cmd.parseUNSUB(line[spaceIndex+1:])
	// case MSG: // e.g. MSG foo 1 5\r\nhello
	// 	return cmd.parseMSG(line[spaceIndex+1:], buffReader)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// parsePUB parses the PUB command with subject and message details
func (c *Cmd) parsePUB(fields []byte, reader *bufio.Reader) (*Cmd, error) {
	parts := bytes.Fields(fields)
	if len(parts) < 2 {
		return nil, fmt.Errorf("PUB command: insufficient arguments")
	}

	c.Subject = string(parts[0])
	bytesLength, err := strconv.Atoi(string(parts[len(parts)-1]))
	if err != nil {
		return nil, fmt.Errorf("PUB command: invalid message length: %w", err)
	}
	if bytesLength < 0 || bytesLength > maxMessageSize {
		return nil, fmt.Errorf("PUB command: message length %d out of bounds", bytesLength)
	}

	c.Bytes = make([]byte, bytesLength)
	if _, err := io.ReadFull(reader, c.Bytes); err != nil {
		return nil, fmt.Errorf("PUB command: failed to read message body: %w", err)
	}

	// Discard the trailing \r\n
	if _, err := reader.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("PUB command: failed to read message body: %w", err)
	}

	return c, nil

}

// parseSUB parses the SUB command with subject, subscription ID, and optional queue group
func (c *Cmd) parseSUB(fields []byte) (*Cmd, error) {
	parts := bytes.Fields(fields)
	if len(parts) < 2 {
		return nil, fmt.Errorf("SUB command: insufficient arguments")
	}
	c.Subject = string(parts[0])
	c.ID = string(parts[1])

	// if len(parts) > 2 {
	// 	c.QueueGroup = parts[2]
	// }

	return c, nil
}

// parseUNSUB parses the UNSUB command with subscription ID and optional max messages
func (c *Cmd) parseUNSUB(fields []byte) (*Cmd, error) {
	parts := bytes.Fields(fields)
	if len(parts) == 0 {
		return nil, fmt.Errorf("UNSUB command: insufficient arguments")
	}
	c.Subject = string(parts[0])

	if len(parts) > 1 {
		maxMessages, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("UNSUB command: invalid max messages: %w", err)
		}
		c.MaxMessagesBeforeUnsub = maxMessages
	}

	return c, nil
}

// parseCONNECT parses the CONNECT command with JSON configuration details
func (c *Cmd) parseCONNECT(payload []byte) (*Cmd, error) {
	var connectData ConnectCommand
	if err := json.Unmarshal(payload, &connectData); err != nil {
		return nil, fmt.Errorf("failed to parse CONNECT command JSON: %w", err)
	}

	c.Name = CONNECT
	c.ConnectData = connectData
	return c, nil
}

// String provides a string representation of a Cmd struct for debugging
func (c *Cmd) String() string {
	return fmt.Sprintf(
		"Name: %s, Subject: %s, ID: %s, Bytes: %s",
		c.Name, c.Subject, c.ID, c.Bytes,
	)
}
