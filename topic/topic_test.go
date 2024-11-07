package topic

import (
	"fmt"
	"sync"
	"testing"
)

type MockClientConnection struct {
	ID       string
	Messages []*Message
	mu       sync.Mutex
}

func NewMockClientConnection(id string) *MockClientConnection {
	return &MockClientConnection{
		ID: id,
	}
}

// Implement the Send method for MockClientConnection
func (c *MockClientConnection) Send(subject string, payload []byte, sid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := &Message{
		Type:    TypeMessage,
		Subject: subject,
		Data:    string(payload),
		ID:      sid,
	}
	c.Messages = append(c.Messages, msg)
	return nil
}

func (c *MockClientConnection) Close() error {
	// Do nothing for mock
	return nil
}

func (c *MockClientConnection) SendError(errMsg, sid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := &Message{
		Type:  TypeError,
		Error: errMsg,
		ID:    sid,
	}
	c.Messages = append(c.Messages, msg)
	return nil
}

// Test adding a subscriber
func TestAddSubscriber(t *testing.T) {
	topic := NewTopic()
	client := NewMockClientConnection("client1")
	sub := &Subscription{
		ClientID: client.ID,
		SID:      "sid1",
		Subject:  "foo.bar",
		Client:   client,
	}

	err := topic.AddSubscriber(sub)
	if err != nil {
		t.Errorf("Failed to add subscriber: %v", err)
	}

	// Try adding the same subscriber again, should return an error
	err = topic.AddSubscriber(sub)
	if err == nil {
		t.Errorf("Expected error when adding duplicate subscriber, but got nil")
	}
}

// Test removing a subscriber
func TestRemoveSubscriber(t *testing.T) {
	topic := NewTopic()
	client := NewMockClientConnection("client1")
	sub := &Subscription{
		ClientID: client.ID,
		SID:      "sid1",
		Subject:  "foo.bar",
		Client:   client,
	}

	// Remove before adding, should return error
	err := topic.RemoveSubscriber(sub.Subject, sub.ClientID)
	if err == nil {
		t.Errorf("Expected error when removing non-existent subscriber, but got nil")
	}

	// Add and then remove
	err = topic.AddSubscriber(sub)
	if err != nil {
		t.Errorf("Failed to add subscriber: %v", err)
	}

	err = topic.RemoveSubscriber(sub.Subject, sub.ClientID)
	if err != nil {
		t.Errorf("Failed to remove subscriber: %v", err)
	}

	// Try removing again, should return error
	err = topic.RemoveSubscriber(sub.Subject, sub.ClientID)
	if err == nil {
		t.Errorf("Expected error when removing already removed subscriber, but got nil")
	}
}

// Test matching and delivering messages
func TestMatchAndDeliver(t *testing.T) {
	topic := NewTopic()
	client1 := NewMockClientConnection("client1")
	client2 := NewMockClientConnection("client2")

	sub1 := &Subscription{
		ClientID: client1.ID,
		SID:      "sid1",
		Subject:  "foo.bar",
		Client:   client1,
	}
	sub2 := &Subscription{
		ClientID: client2.ID,
		SID:      "sid2",
		Subject:  "foo.*",
		Client:   client2,
	}

	err := topic.AddSubscriber(sub1)
	if err != nil {
		t.Errorf("Failed to add subscriber 1: %v", err)
	}
	err = topic.AddSubscriber(sub2)
	if err != nil {
		t.Errorf("Failed to add subscriber 2: %v", err)
	}

	msg := []byte("hello world")
	subscribers, err := topic.MatchAndDeliver("foo.bar", msg)
	if err != nil {
		t.Errorf("Failed to match and deliver: %v", err)
	}
	if len(subscribers) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", len(subscribers))
	}

	// Check if clients received the message
	if len(client1.Messages) != 1 {
		t.Errorf("Client 1 should have received 1 message, got %d", len(client1.Messages))
	}
	if len(client2.Messages) != 1 {
		t.Errorf("Client 2 should have received 1 message, got %d", len(client2.Messages))
	}
}

// Test wildcard subscriptions
func TestWildcardSubscriptions(t *testing.T) {
	topic := NewTopic()
	client1 := NewMockClientConnection("client1")
	client2 := NewMockClientConnection("client2")
	client3 := NewMockClientConnection("client3")

	sub1 := &Subscription{
		ClientID: client1.ID,
		SID:      "sid1",
		Subject:  "foo.*",
		Client:   client1,
	}
	sub2 := &Subscription{
		ClientID: client2.ID,
		SID:      "sid2",
		Subject:  "foo.>",
		Client:   client2,
	}
	sub3 := &Subscription{
		ClientID: client3.ID,
		SID:      "sid3",
		Subject:  "foo.bar.baz",
		Client:   client3,
	}

	err := topic.AddSubscriber(sub1)
	if err != nil {
		t.Errorf("Failed to add subscriber 1: %v", err)
	}
	err = topic.AddSubscriber(sub2)
	if err != nil {
		t.Errorf("Failed to add subscriber 2: %v", err)
	}
	err = topic.AddSubscriber(sub3)
	if err != nil {
		t.Errorf("Failed to add subscriber 3: %v", err)
	}

	msg := []byte("hello wildcard")
	subscribers, err := topic.MatchAndDeliver("foo.bar.baz", msg)
	if err != nil {
		t.Errorf("Failed to match and deliver: %v", err)
	}
	if len(subscribers) != 2 {
		t.Errorf("Expected 2 subscribers (client2 and client3), got %d", len(subscribers))
	}

	if len(client1.Messages) != 0 {
		t.Errorf("Client 1 should not have received any messages, got %d", len(client1.Messages))
	}
	if len(client2.Messages) != 1 {
		t.Errorf("Client 2 should have received 1 message, got %d", len(client2.Messages))
	}
	if len(client3.Messages) != 1 {
		t.Errorf("Client 3 should have received 1 message, got %d", len(client3.Messages))
	}
}

// Test handling unsubscription with MaxMsgs
func TestHandleUnsubWithMaxMsgs(t *testing.T) {
	topic := NewTopic()
	client := NewMockClientConnection("client1")

	sub := &Subscription{
		ClientID: client.ID,
		SID:      "sid1",
		Subject:  "foo.bar",
		Client:   client,
		MaxMsgs:  3, // Should receive 3 messages before unsubscribing
	}

	err := topic.AddSubscriber(sub)
	if err != nil {
		t.Errorf("Failed to add subscriber: %v", err)
	}

	msg := []byte("test message")
	for i := 0; i < 5; i++ {
		_, err := topic.MatchAndDeliver("foo.bar", msg)
		if err != nil {
			t.Errorf("Failed to match and deliver: %v", err)
		}
	}

	// Client should have received only 3 messages
	if len(client.Messages) != 3 {
		t.Errorf("Client should have received 3 messages, got %d", len(client.Messages))
	}

	// Subscription should have been removed
	current := topic.getTopicNodeSubject("foo.bar")
	if current != nil {
		t.Errorf("Topic node for 'foo.bar' should exist")
		current.mu.RLock()
		_, exists := current.subscribers[client.ID]
		current.mu.RUnlock()
		if exists {
			t.Errorf("Subscriber should have been removed after MaxMsgs reached")
		}
	}
}

// Test concurrent adding and removing of subscribers
func TestConcurrentAddAndRemoveSubscribers(t *testing.T) {
	topic := NewTopic()
	numClients := 100
	numMessages := 10
	clients := make([]*MockClientConnection, numClients)
	subs := make([]*Subscription, numClients)

	// Add subscribers concurrently
	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sid := fmt.Sprintf("client-%d", i)
			client := NewMockClientConnection(sid)
			sub := &Subscription{
				ClientID: client.ID,
				SID:      sid,
				Subject:  "foo.bar",
				Client:   client,
			}
			err := topic.AddSubscriber(sub)
			if err != nil {
				t.Errorf("Failed to add subscriber %d: %v", i, err)
			}
			clients[i] = client
			subs[i] = sub
		}(i)
	}
	wg.Wait()

	// Publish messages concurrently
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := topic.MatchAndDeliver("foo.bar", []byte("message"))
			if err != nil {
				t.Errorf("Failed to match and deliver: %v", err)
			}
		}()
	}
	wg.Wait()

	// Check that all clients received messages
	for i, client := range clients {
		if len(client.Messages) != numMessages {
			t.Errorf("Client %d should have received %d messages, got %d", i, numMessages, len(client.Messages))
		}
	}

	// Remove subscribers concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := topic.RemoveSubscriber(subs[i].Subject, subs[i].ClientID)
			if err != nil {
				t.Errorf("Failed to remove subscriber %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
}

// Test adding subscribers with invalid subjects
func TestInvalidSubjects(t *testing.T) {
	topic := NewTopic()
	client := NewMockClientConnection("client1")

	invalidSubjects := []string{
		"",
		"foo..bar",
		"foo.bar.",
		".foo.bar",
		"foo.>.*",
		"foo.>.bar",
		"foo.>.>",
	}

	for _, subject := range invalidSubjects {
		sub := &Subscription{
			ClientID: client.ID,
			SID:      "sid1",
			Subject:  subject,
			Client:   client,
		}
		err := topic.AddSubscriber(sub)
		if err == nil {
			t.Errorf("Expected error when adding subscriber with invalid subject '%s', but got nil", subject)
		}
	}
}

// Test subject validation
func TestSubjectValidation(t *testing.T) {
	validSubjects := []string{
		"foo",
		"foo.bar",
		"foo.bar.baz",
		"foo.*",
		"foo.bar.*",
		"foo.>",
		"foo.bar.>",
	}

	for _, subject := range validSubjects {
		err := validateSubject(subject)
		if err != nil {
			t.Errorf("Expected subject '%s' to be valid, but got error: %v", subject, err)
		}
	}

	invalidSubjects := []string{
		"",
		"foo..bar",
		"foo.bar.",
		".foo.bar",
		"foo.>.*",
		"foo.>.bar",
		"foo.>.>",
	}

	for _, subject := range invalidSubjects {
		err := validateSubject(subject)
		if err == nil {
			t.Errorf("Expected subject '%s' to be invalid, but got no error", subject)
		}
	}
}
