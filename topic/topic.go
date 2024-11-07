package topic

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Subscription struct {
	ClientID string // Client ID for the connection
	SID      string // Subscription ID for tracking
	Subject  string
	MaxMsgs  int32 // Maximum messages to receive before unsubscribing
	Client   IClientConnection
}

type Topic struct {
	root  *node
	mutex sync.Mutex
}

type node struct {
	children map[string]*node
	// every subscriber is unique by ID
	subscribers map[string]*Subscription // TODO: Use map of subscriptiona array for multiple subscribers with different SIDs
	mu          sync.RWMutex
}

func NewTrieNode() *node {
	return &node{
		children:    make(map[string]*node),
		subscribers: make(map[string]*Subscription),
	}
}

func NewTopic() *Topic {
	return &Topic{
		root: NewTrieNode(),
	}
}

// validateSubject ensures the subject format is valid
func validateSubject(subject string) error {
	if subject == "" {
		return fmt.Errorf("empty subject")
	}
	parts := strings.Split(subject, ".")
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("empty token in subject at position %d", i)
		}
		if part == ">" && i != len(parts)-1 {
			return fmt.Errorf("'>' wildcard must be last token")
		}
		if strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("invalid character in subject token at position %d", i)
		}
	}
	return nil
}

// AddSubscriber adds a subscription to the trie based on the subject
func (t *Topic) AddSubscriber(sub *Subscription) error {
	if sub == nil {
		return fmt.Errorf("nil subscription")
	}
	if err := validateSubject(sub.Subject); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	parts := splitSubject(sub.Subject)
	current := t.root
	current.mu.Lock()

	for _, part := range parts {
		child, exists := current.children[part]
		if !exists {
			child = NewTrieNode()
			current.children[part] = child
		}
		// Lock the child before unlocking the parent
		child.mu.Lock()
		current.mu.Unlock()
		current = child
	}

	defer current.mu.Unlock()
	if _, exists := current.subscribers[sub.ClientID]; exists {
		return fmt.Errorf("subscription with ID %s already exists for subject %s", sub.ClientID, sub.Subject)
	}
	current.subscribers[sub.ClientID] = sub
	return nil
}

// RemoveSubscriber removes a subscription by ID from the specified subject path
func (t *Topic) RemoveSubscriber(subject, clientID string) error {
	if err := validateSubject(subject); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	parts := splitSubject(subject)
	current := t.root
	var nodePath []*node
	current.mu.Lock()

	for _, part := range parts {
		child, exists := current.children[part]
		if !exists {
			current.mu.Unlock()
			return fmt.Errorf("subject path %s does not exist", subject)
		}
		// Lock the child before unlocking the parent
		child.mu.Lock()
		nodePath = append(nodePath, current)
		current.mu.Unlock()
		current = child
	}

	// Now current.mu is locked for the target node
	defer current.mu.Unlock()

	sub, exists := current.subscribers[clientID]
	if !exists {
		return fmt.Errorf("subscriber with ID %s not found for subject %s", clientID, subject)
	}

	// Remove the subscriber
	sub.Client.Close()
	delete(current.subscribers, clientID)

	// Check if node is empty and clean up if necessary
	isEmpty := len(current.subscribers) == 0 && len(current.children) == 0
	if !isEmpty {
		return nil
	}

	// Cleanup empty nodes
	for i := len(parts) - 1; i >= 0; i-- {
		parent := nodePath[i]
		part := parts[i]
		parent.mu.Lock()
		delete(parent.children, part)
		isEmpty = len(parent.subscribers) == 0 && len(parent.children) == 0
		parent.mu.Unlock()
		// fmt.Println("Removed node", parts[i])
		if !isEmpty {
			break
		}
	}
	// fmt.Println("Removed subscriber", clientID, "from subject", subject)
	return nil
}

// MatchAndDeliver finds all matching subscribers and delivers the message to them
func (t *Topic) MatchAndDeliver(subject string, msg []byte) ([]*Subscription, error) {
	if err := validateSubject(subject); err != nil {
		return nil, fmt.Errorf("invalid subject: %w", err)
	}

	parts := splitSubject(subject)
	subscribers := make([]*Subscription, 0)
	t.matchHelper(t.root, parts, 0, &subscribers)

	// Deliver messages concurrently
	var wg sync.WaitGroup
	var errs []error
	var errMu sync.Mutex

	for _, sub := range subscribers {
		wg.Add(1)
		go func(s *Subscription) {
			defer wg.Done()
			if err := s.Client.Send(subject, msg, s.SID); err != nil {
				errMu.Lock()
				errs = append(errs, fmt.Errorf("failed to deliver to client %s: %w", s.ClientID, err))
				errMu.Unlock()
			}
			// Decrement max messages and remove subscriber if limit reached
			if s.MaxMsgs > 0 {
				remaining := atomic.AddInt32(&s.MaxMsgs, -1)
				if remaining == 0 {
					fmt.Println("Unsubscribing client", s.ClientID, "from subject", s.Subject)
					if err := t.RemoveSubscriber(s.Subject, s.ClientID); err != nil {
						errMu.Lock()
						errs = append(errs, fmt.Errorf("failed to remove subscriber %s: %w", s.ClientID, err))
						errMu.Unlock()
					}
				}
			}
		}(sub)
	}
	wg.Wait()

	if len(errs) > 0 {
		return subscribers, fmt.Errorf("delivery errors occurred: %v", errs)
	}

	return subscribers, nil

}

// matchHelper recursively matches subjects with subscribers in the trie
func (t *Topic) matchHelper(current *node, parts []string, index int, subscribers *[]*Subscription) {
	if current == nil {
		return
	}
	current.mu.RLock()
	defer current.mu.RUnlock()

	if index == len(parts) {
		for _, sub := range current.subscribers {
			*subscribers = append(*subscribers, sub)
		}
		return
	}

	if index > len(parts) {
		return // Prevent potential index out of bounds
	}

	part := parts[index]

	// Check for exact match
	if next, exists := current.children[part]; exists {
		t.matchHelper(next, parts, index+1, subscribers)
	}

	// Check for single-level wildcard
	if next, exists := current.children["*"]; exists {
		t.matchHelper(next, parts, index+1, subscribers)
	}

	// Check for multi-level wildcard
	if next, exists := current.children[">"]; exists {
		for _, sub := range next.subscribers {
			*subscribers = append(*subscribers, sub)
		}
	}
}

func (t *Topic) HanlderUnsub(subject string, clientID string, maxMsg int) error {
	if err := validateSubject(subject); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	current := t.getTopicNodeSubject(subject)

	if current == nil {
		return fmt.Errorf("subject %s does not exist", subject)
	}

	current.mu.Lock()
	defer current.mu.Unlock()
	sub, exists := current.subscribers[clientID]
	if !exists {
		return fmt.Errorf("subscriber with ID %s not found for subject %s", clientID, subject)
	}

	if maxMsg > 0 {
		atomic.StoreInt32(&sub.MaxMsgs, int32(maxMsg))
	} else {
		// Remove subscriber
		delete(current.subscribers, clientID)
		// TODO, check if the node is empty and clean up
	}

	return nil

}

// getTopicNodeSubject traverses the trie to find the node corresponding to the subject
func (t *Topic) getTopicNodeSubject(subject string) *node {
	parts := splitSubject(subject)
	current := t.root

	for _, part := range parts {
		current.mu.RLock()
		child, exists := current.children[part]
		current.mu.RUnlock()
		if !exists {
			return nil
		}
		current = child
	}
	return current
}

func splitSubject(subject string) []string {
	return strings.Split(subject, ".")
}

func (t *Topic) PrintTopic() {
	for {
		time.Sleep(10 * time.Second)
		t.printHelper(t.root, "")
	}
}

func (t *Topic) printHelper(current *node, indent string) {
	if current == nil {
		return
	}
	current.mu.RLock()
	defer current.mu.RUnlock()

	for key, child := range current.children {
		fmt.Printf("%s%s\n", indent, key)
		t.printHelper(child, indent+"  ")
	}
}
