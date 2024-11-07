package topic

import "time"

type MessageType string

const (
	// Server -> Client messages
	TypeMessage MessageType = "MSG"   // Regular message
	TypeAck     MessageType = "ACK"   // Acknowledgment
	TypeError   MessageType = "ERROR" // Error notification

	// Client -> Server messages
	TypeSubscribe   MessageType = "SUB"   // Subscribe request
	TypeUnsubscribe MessageType = "UNSUB" // Unsubscribe request
	TypePublish     MessageType = "PUB"   // Publish message
)

// Message represents the structure of messages sent between server and clients
type Message struct {
	Type      MessageType `json:"type"`
	Subject   string      `json:"subject"`
	Data      string      `json:"data,omitempty"`
	ID        string      `json:"id"` // Message ID for tracking
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}
