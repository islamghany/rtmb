package commands

import (
	"fmt"
	"net"

	"github.com/islamghany/rtmb/parser"
	"github.com/islamghany/rtmb/topic"
)

type Commander struct {
	conn       net.Conn
	topic      *topic.Topic
	clientID   string
	clientSubs map[string]bool // map of client subject to subscription
	port       string
}

type CommanderConfig struct {
	Conn  net.Conn
	Topic *topic.Topic
	Port  string
}

// NewCommander initializes a Commander with configuration
func NewCommander(cfg *CommanderConfig) *Commander {
	return &Commander{
		conn:       cfg.Conn,
		topic:      cfg.Topic,
		clientSubs: make(map[string]bool),
		clientID:   cfg.Conn.RemoteAddr().String(),
		port:       cfg.Port,
	}
}

// HandleCommand processes each command parsed from the connection
func (c *Commander) HandleCommand(cmd *parser.Cmd) error {
	switch cmd.Name {
	case parser.CONNECT:
		return c.handleConnect(cmd)
	case parser.PONG:
		return nil
	case parser.SUB:
		return c.handleSub(cmd)
	case parser.PUB:
		return c.handlePub(cmd)
	case parser.UNSUB:
		return c.hanlderUnsub(cmd)
	default:
		return fmt.Errorf("Unknown Command: %s", cmd.Name)
	}
}

// handleConnect processes the CONNECT command
func (c *Commander) handleConnect(cmd *parser.Cmd) error {
	// Process connection options if needed
	// Example: c.verbose = cmd.ConnectData.Verbose
	return c.sendOK()
}

// handleSub processes the SUB command, adding a new subscriber to the topic
func (c *Commander) handleSub(cmd *parser.Cmd) error {
	client := topic.NewClientConnection(c.clientID, c.conn)
	sub := &topic.Subscription{
		ClientID: c.clientID,
		SID:      cmd.ID,
		Subject:  cmd.Subject,
		Client:   client,
	}

	if err := c.topic.AddSubscriber(sub); err != nil {
		c.sendError(fmt.Sprintf("failed to add subscriber: %v", err))
		client.Close()
		return err
	}

	// Track subscription
	c.clientSubs[cmd.Subject] = true

	return c.sendOK()
}

// handleSub processes the SUB command, adding a new subscriber to the topic
func (c *Commander) handlePub(cmd *parser.Cmd) error {
	subscribers, err := c.topic.MatchAndDeliver(cmd.Subject, cmd.Bytes)
	if err != nil {
		return c.sendError(fmt.Sprintf("failed to deliver message: %v", err))
	}

	// If no subscribers, acknowledge success without delivery
	if len(subscribers) == 0 {
		fmt.Println("No subscribers matched for subject:", cmd.Subject)
	}
	return c.sendOK()
}

// hanlderUnsub processes the UNSUB command, removing a subscriber from the topic
func (c *Commander) hanlderUnsub(cmd *parser.Cmd) error {
	if _, ok := c.clientSubs[cmd.Subject]; !ok {
		return c.sendError(fmt.Sprintf("subject %s not found in you subscriptions", cmd.Subject))
	}

	if err := c.topic.HanlderUnsub(
		cmd.Subject,
		c.clientID,
		cmd.MaxMessagesBeforeUnsub,
	); err != nil {
		return c.sendError(fmt.Sprintf("failed to remove subscriber: %v", err))
	}

	delete(c.clientSubs, cmd.Subject)

	return c.sendOK()
}

// sendOK writes an acknowledgment to the client
func (c *Commander) sendOK() error {
	_, err := c.conn.Write([]byte("+OK\r\n"))
	return err
}

// sendError writes an error message to the client with optional info
func (c *Commander) sendError(info string) error {
	msg := fmt.Sprintf("-ERR '%s'\r\n", info)
	_, err := c.conn.Write([]byte(msg))
	return err
}
func (c *Commander) SendInfo() error {
	// Static info for now
	info := fmt.Sprintf("INFO {\"server_id\":\"rtmb-1.0\",\"version\":\"1.0.0\",\"proto\":\"tcp\",\"host\":\"localhost\",\"port\":%s,\"auth_required\":false,\"ssl_required\":false,\"ssl_verify\":false,\"max_payload\":1048576}\n", c.port)
	_, err := c.conn.Write([]byte(info))
	return err
}

func (c *Commander) CleanupSubscriptions() {
	for subject := range c.clientSubs {
		c.topic.RemoveSubscriber(subject, c.clientID)
	}
}
