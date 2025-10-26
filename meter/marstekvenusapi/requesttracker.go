package marstekvenusapi

import "sync"

// RequestTracker stores the request type associated with a sent ID.
type RequestTracker struct {
	sync.Mutex      // Mutex protects nextID and pendingRequests
	nextID          int
	pendingRequests map[int]RequestType // Map ID to RequestType
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		nextID:          1001, // Starting ID
		pendingRequests: make(map[int]RequestType),
	}
}

// GetNextID increments the counter and returns the new ID safely.
func (rt *RequestTracker) GetNextID() int {
	rt.Lock()
	defer rt.Unlock()
	rt.nextID++
	return rt.nextID
}

// TrackRequest records the type of request sent.
func (rt *RequestTracker) TrackRequest(id int, reqType RequestType) {
	rt.Lock()
	defer rt.Unlock()
	rt.pendingRequests[id] = reqType
}

// RetrieveRequestType retrieves the type and cleans up the tracking map.
func (rt *RequestTracker) RetrieveRequestType(id int) (RequestType, bool) {
	rt.Lock()
	defer rt.Unlock()
	reqType, exists := rt.pendingRequests[id]
	if exists {
		delete(rt.pendingRequests, id)
	}
	return reqType, exists
}
