package ocpp

import (
	"net"
	"net/http"
	"sync"

	"github.com/lorenzodonini/ocpp-go/ws"
)

var (
	globalMux   *sharedMux
	globalMuxMu sync.Mutex
)

type sharedMux struct {
	mu        sync.RWMutex
	handler16 http.Handler
	handler20 http.Handler
	tcpAddr   *net.TCPAddr
}

func (m *sharedMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, proto := range r.Header["Sec-Websocket-Protocol"] {
		if proto == "ocpp2.0.1" {
			m.mu.RLock()
			h := m.handler20
			m.mu.RUnlock()
			if h != nil {
				h.ServeHTTP(w, r)
				return
			}
			http.Error(w, "OCPP 2.0.1 not configured", http.StatusServiceUnavailable)
			return
		}
	}
	m.mu.RLock()
	h := m.handler16
	m.mu.RUnlock()
	if h != nil {
		h.ServeHTTP(w, r)
		return
	}
	http.Error(w, "OCPP 1.6 not configured", http.StatusServiceUnavailable)
}

// initSharedMux creates the shared HTTP mux and registers the OCPP 1.6 handler.
// Called from listen() after the TCP listener is bound.
func initSharedMux(server16 ws.Server, path string, tcpAddr *net.TCPAddr) *sharedMux {
	m := &sharedMux{
		tcpAddr:   tcpAddr,
		handler16: server16.HTTPHandler(path),
	}
	server16.SetAddr(tcpAddr)
	globalMuxMu.Lock()
	globalMux = m
	globalMuxMu.Unlock()
	return m
}

// RegisterMuxServer20 registers the OCPP 2.0.1 ws.Server with the shared mux.
// Called from ocpp20.Instance() after ocpp.Instance() has started the listener.
func RegisterMuxServer20(server20 ws.Server, path string) {
	globalMuxMu.Lock()
	m := globalMux
	globalMuxMu.Unlock()
	if m == nil {
		return
	}
	server20.SetAddr(m.tcpAddr)
	m.mu.Lock()
	m.handler20 = server20.HTTPHandler(path)
	m.mu.Unlock()
}
