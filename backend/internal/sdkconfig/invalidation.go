package sdkconfig

import "sync"

type Invalidation struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id,omitempty"`
	CredentialID  string `json:"credential_id,omitempty"`
}

type InvalidationHub struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]invalidationSubscriber
}

type invalidationSubscriber struct {
	credential Credential
	channel    chan Invalidation
}

func NewInvalidationHub() *InvalidationHub {
	return &InvalidationHub{subscribers: make(map[uint64]invalidationSubscriber)}
}

func (h *InvalidationHub) Subscribe(credential Credential) (<-chan Invalidation, func()) {
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	channel := make(chan Invalidation, 1)
	h.subscribers[id] = invalidationSubscriber{credential: credential, channel: channel}
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers, id)
			close(channel)
			h.mu.Unlock()
		})
	}
	return channel, cancel
}

func (h *InvalidationHub) Publish(invalidation Invalidation) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, subscriber := range h.subscribers {
		if !invalidationMatches(subscriber.credential, invalidation) {
			continue
		}
		select {
		case subscriber.channel <- invalidation:
		default:
			// Configuration invalidations coalesce. One pending signal is enough to
			// make an SDK conditionally refresh to the latest source-of-truth state.
		}
	}
}

func invalidationMatches(credential Credential, invalidation Invalidation) bool {
	if invalidation.CredentialID != "" {
		return credential.ID == invalidation.CredentialID
	}
	if credential.ProjectID != invalidation.ProjectID {
		return false
	}
	return invalidation.EnvironmentID == "" || credential.EnvironmentID == invalidation.EnvironmentID
}
