package topic

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

type IClientConnection interface {
	Send(subject string, payload []byte, sid string) error
	SendError(errMsg, sid string) error
	Close() error
}

type ClientConnection struct {
	ID        string
	Conn      io.Writer
	mu        sync.Mutex
	writeChan chan *Message // Channel for async message writing
	done      chan struct{} // Channel to signal shutdown
}

func NewClientConnection(id string, conn io.Writer) *ClientConnection {
	client := &ClientConnection{
		ID:        id,
		Conn:      conn,
		writeChan: make(chan *Message, 100), // Buffered channel for messages
		done:      make(chan struct{}),      // Channel to signal shutdown
	}
	// Start the write loop
	go client.writeLoop()

	return client
}

func (c *ClientConnection) writeLoop() {
	for {
		select {
		case msg, ok := <-c.writeChan:
			if !ok {
				// Channel closed, exit the loop
				return
			}
			if err := c.writeMessage(msg); err != nil {
				log.Printf("Error writing to client %s: %v\n", c.ID, err)
				// Close the client on write error
				c.Close()
				return
			}
		case <-c.done:
			// Drain remaining messages before exiting
			for msg := range c.writeChan {
				if err := c.writeMessage(msg); err != nil {
					log.Printf("Error writing to client %s: %v\n", c.ID, err)
				}
			}
			return
		}
	}
}

func (c *ClientConnection) writeMessage(msg *Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Add CRLF as per protocol
	data = append(data, '\r', '\n')

	_, err = c.Conn.Write(data)
	return err
}

// Send now uses the message channel for async writing
func (c *ClientConnection) Send(subject string, payload []byte, sid string) error {
	msg := &Message{
		Type:      TypeMessage,
		Subject:   subject,
		Data:      string(payload),
		ID:        sid,
		Timestamp: time.Now(),
	}

	select {
	case c.writeChan <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("client write buffer full for client %s", c.ID)
	}
}

// SendError sends an error message to the client
func (c *ClientConnection) SendError(errMsg, sid string) error {
	msg := &Message{
		Type:      TypeError,
		Error:     errMsg,
		ID:        sid,
		Timestamp: time.Now(),
	}

	select {
	case c.writeChan <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("client write buffer full for client %s", c.ID)
	}
}

// Close gracefully shuts down the client connection
func (c *ClientConnection) Close() error {
	// Signal writeLoop to exit
	close(c.done)
	// Close writeChan to stop receiving new messages
	close(c.writeChan)
	c.mu.Lock()
	defer c.mu.Unlock()
	return nil
}
