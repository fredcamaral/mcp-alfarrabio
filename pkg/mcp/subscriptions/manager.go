package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager manages subscriptions for MCP
type Manager struct {
	subscriptions map[string]*Subscription
	byClient      map[string][]string // clientID -> subscriptionIDs
	byMethod      map[string][]string // method -> subscriptionIDs
	eventChan     chan Event
	handlers      map[string]EventHandler
	mutex         sync.RWMutex
	maxPerClient  int
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// EventHandler processes events for a specific subscription type
type EventHandler func(ctx context.Context, event Event) error

// NewManager creates a new subscription manager
func NewManager(maxPerClient int) *Manager {
	return &Manager{
		subscriptions: make(map[string]*Subscription),
		byClient:      make(map[string][]string),
		byMethod:      make(map[string][]string),
		eventChan:     make(chan Event, 1000),
		handlers:      make(map[string]EventHandler),
		maxPerClient:  maxPerClient,
	}
}

// Start starts the subscription manager
func (m *Manager) Start(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.wg.Add(1)
	go m.processEvents()
}

// Stop stops the subscription manager
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// Subscribe creates a new subscription
func (m *Manager) Subscribe(clientID string, method string, params json.RawMessage) (*Subscription, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check client subscription limit
	if len(m.byClient[clientID]) >= m.maxPerClient {
		return nil, fmt.Errorf("client %s has reached maximum subscription limit (%d)", clientID, m.maxPerClient)
	}

	// Validate method
	if !m.isValidMethod(method) {
		return nil, fmt.Errorf("unsupported subscription method: %s", method)
	}

	// Create subscription
	sub := &Subscription{
		ID:         uuid.New().String(),
		ClientID:   clientID,
		Method:     method,
		Params:     params,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	// Store subscription
	m.subscriptions[sub.ID] = sub
	m.byClient[clientID] = append(m.byClient[clientID], sub.ID)
	m.byMethod[method] = append(m.byMethod[method], sub.ID)

	return sub, nil
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(subscriptionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sub, exists := m.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// Remove from all indices
	delete(m.subscriptions, subscriptionID)
	
	// Remove from client index
	clientSubs := m.byClient[sub.ClientID]
	m.removeFromSlice(&clientSubs, subscriptionID)
	m.byClient[sub.ClientID] = clientSubs
	if len(m.byClient[sub.ClientID]) == 0 {
		delete(m.byClient, sub.ClientID)
	}
	
	// Remove from method index
	methodSubs := m.byMethod[sub.Method]
	m.removeFromSlice(&methodSubs, subscriptionID)
	m.byMethod[sub.Method] = methodSubs
	if len(m.byMethod[sub.Method]) == 0 {
		delete(m.byMethod, sub.Method)
	}

	return nil
}

// UnsubscribeClient removes all subscriptions for a client
func (m *Manager) UnsubscribeClient(clientID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	subIDs, exists := m.byClient[clientID]
	if !exists {
		return nil
	}

	// Remove all subscriptions for this client
	for _, subID := range subIDs {
		if sub, exists := m.subscriptions[subID]; exists {
			delete(m.subscriptions, subID)
			methodSubs := m.byMethod[sub.Method]
			m.removeFromSlice(&methodSubs, subID)
			m.byMethod[sub.Method] = methodSubs
			if len(m.byMethod[sub.Method]) == 0 {
				delete(m.byMethod, sub.Method)
			}
		}
	}

	delete(m.byClient, clientID)
	return nil
}

// GetSubscription returns a subscription by ID
func (m *Manager) GetSubscription(subscriptionID string) (*Subscription, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sub, exists := m.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	return sub, nil
}

// ListSubscriptions returns all subscriptions for a client
func (m *Manager) ListSubscriptions(clientID string) []*Subscription {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	subIDs, exists := m.byClient[clientID]
	if !exists {
		return []*Subscription{}
	}

	subs := make([]*Subscription, 0, len(subIDs))
	for _, subID := range subIDs {
		if sub, exists := m.subscriptions[subID]; exists {
			subs = append(subs, sub)
		}
	}

	return subs
}

// PublishEvent publishes an event to relevant subscribers
func (m *Manager) PublishEvent(eventType string, data interface{}) {
	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case m.eventChan <- event:
	default:
		// Event channel full, drop event
		// In production, you'd want better handling here
	}
}

// RegisterHandler registers an event handler for a subscription method
func (m *Manager) RegisterHandler(method string, handler EventHandler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.handlers[method] = handler
}

// processEvents processes events in the background
func (m *Manager) processEvents() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case event := <-m.eventChan:
			m.handleEvent(event)
		}
	}
}

// handleEvent handles a single event
func (m *Manager) handleEvent(event Event) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Determine which subscriptions should receive this event
	method := m.eventTypeToMethod(event.Type)
	if method == "" {
		return
	}

	subIDs, exists := m.byMethod[method]
	if !exists {
		return
	}

	// Send event to each matching subscription
	for _, subID := range subIDs {
		sub, exists := m.subscriptions[subID]
		if !exists {
			continue
		}

		// Check if subscription matches event
		if m.matchesSubscription(sub, event) {
			event.SubscriptionID = subID
			
			// Call handler if registered
			if handler, exists := m.handlers[method]; exists {
				go handler(m.ctx, event)
			}
		}
	}
}

// matchesSubscription checks if an event matches a subscription's criteria
func (m *Manager) matchesSubscription(sub *Subscription, event Event) bool {
	// Parse subscription parameters
	switch sub.Method {
	case "resources/subscribe":
		var params ResourceSubscriptionParams
		if err := json.Unmarshal(sub.Params, &params); err != nil {
			return false
		}
		
		// Check if event is for this resource
		if changeEvent, ok := event.Data.(*ResourceChangeEvent); ok {
			return changeEvent.URI == params.URI
		}
		
	case "tools/subscribe", "resources/subscribeList", "prompts/subscribe", "roots/subscribe":
		// List change subscriptions match all events of their type
		return true
		
	default:
		return false
	}
	
	return false
}

// eventTypeToMethod maps event types to subscription methods
func (m *Manager) eventTypeToMethod(eventType string) string {
	switch eventType {
	case EventTypeResourceChanged, EventTypeResourceDeleted:
		return "resources/subscribe"
	case EventTypeToolsListChanged:
		return "tools/subscribe"
	case EventTypeResourcesListChanged:
		return "resources/subscribeList"
	case EventTypePromptsListChanged:
		return "prompts/subscribe"
	case EventTypeRootsListChanged:
		return "roots/subscribe"
	default:
		return ""
	}
}

// isValidMethod checks if a subscription method is valid
func (m *Manager) isValidMethod(method string) bool {
	validMethods := []string{
		"resources/subscribe",
		"resources/subscribeList",
		"tools/subscribe",
		"prompts/subscribe",
		"roots/subscribe",
	}
	
	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	
	return false
}

// removeFromSlice removes an item from a slice
func (m *Manager) removeFromSlice(slice *[]string, item string) {
	for i, v := range *slice {
		if v == item {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
}

// TouchSubscription updates the last active time for a subscription
func (m *Manager) TouchSubscription(subscriptionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if sub, exists := m.subscriptions[subscriptionID]; exists {
		sub.LastActive = time.Now()
	}
}

// CleanupInactive removes subscriptions that haven't been active
func (m *Manager) CleanupInactive(maxInactive time.Duration) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-maxInactive)
	removed := 0

	for subID, sub := range m.subscriptions {
		if sub.LastActive.Before(cutoff) {
			delete(m.subscriptions, subID)
			clientSubs := m.byClient[sub.ClientID]
			m.removeFromSlice(&clientSubs, subID)
			m.byClient[sub.ClientID] = clientSubs
			methodSubs := m.byMethod[sub.Method]
			m.removeFromSlice(&methodSubs, subID)
			m.byMethod[sub.Method] = methodSubs
			removed++
		}
	}

	// Clean up empty indices
	for clientID, subs := range m.byClient {
		if len(subs) == 0 {
			delete(m.byClient, clientID)
		}
	}
	
	for method, subs := range m.byMethod {
		if len(subs) == 0 {
			delete(m.byMethod, method)
		}
	}

	return removed
}