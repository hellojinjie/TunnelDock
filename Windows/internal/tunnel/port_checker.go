package tunnel

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type PortStatus int

const (
	PortAvailable PortStatus = iota
	PortUsedByTunnelDock
	PortUsedExternally
)

type Endpoint struct {
	Address string
	Port    uint16
}

type PortChecker struct {
	mu           sync.Mutex
	reservations map[string]struct{}
}

func NewPortChecker() *PortChecker {
	return &PortChecker{reservations: make(map[string]struct{})}
}

func (c *PortChecker) Check(endpoint Endpoint) PortStatus {
	key := endpointKey(endpoint)
	c.mu.Lock()
	_, reserved := c.reservations[key]
	c.mu.Unlock()
	if reserved {
		return PortUsedByTunnelDock
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(unbracket(endpoint.Address), fmt.Sprintf("%d", endpoint.Port)))
	if err != nil {
		return PortUsedExternally
	}
	_ = listener.Close()
	return PortAvailable
}

func (c *PortChecker) Reserve(endpoint Endpoint) bool {
	key := endpointKey(endpoint)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.reservations[key]; exists {
		return false
	}
	c.reservations[key] = struct{}{}
	return true
}

func (c *PortChecker) Release(endpoint Endpoint) {
	c.mu.Lock()
	delete(c.reservations, endpointKey(endpoint))
	c.mu.Unlock()
}

func endpointKey(endpoint Endpoint) string {
	address := strings.ToLower(unbracket(strings.TrimSpace(endpoint.Address)))
	if strings.EqualFold(address, "localhost") {
		address = "loopback"
	} else if parsed := net.ParseIP(address); parsed != nil && parsed.IsLoopback() {
		address = "loopback"
	}
	return fmt.Sprintf("%s:%d", address, endpoint.Port)
}

func unbracket(value string) string {
	return strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
}
