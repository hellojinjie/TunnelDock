package tunnel

import (
	"net"
	"strconv"
	"testing"
)

func TestPortCheckerReportsAvailableAndExternalOccupancy(t *testing.T) {
	checker := NewPortChecker()
	available := unusedEndpoint(t)
	if got := checker.Check(available); got != PortAvailable {
		t.Fatalf("Check(unused) = %v, want PortAvailable", got)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	occupied := endpointFromListener(t, listener)
	if got := checker.Check(occupied); got != PortUsedExternally {
		t.Fatalf("Check(external) = %v, want PortUsedExternally", got)
	}
}

func TestPortCheckerReservationDistinguishesTunnelDockAndNormalizesLoopback(t *testing.T) {
	checker := NewPortChecker()
	endpoint := unusedEndpoint(t)
	if !checker.Reserve(endpoint) {
		t.Fatal("first Reserve() = false")
	}
	equivalent := Endpoint{Address: "localhost", Port: endpoint.Port}
	if got := checker.Check(equivalent); got != PortUsedByTunnelDock {
		t.Fatalf("Check(equivalent loopback) = %v, want PortUsedByTunnelDock", got)
	}
	if checker.Reserve(equivalent) {
		t.Fatal("second equivalent Reserve() = true")
	}
	checker.Release(Endpoint{Address: "::1", Port: endpoint.Port})
	if got := checker.Check(endpoint); got != PortAvailable {
		t.Fatalf("Check() after equivalent release = %v, want PortAvailable", got)
	}
}

func unusedEndpoint(t *testing.T) Endpoint {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := endpointFromListener(t, listener)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return endpoint
}

func endpointFromListener(t *testing.T, listener net.Listener) Endpoint {
	t.Helper()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return Endpoint{Address: host, Port: uint16(port)}
}
