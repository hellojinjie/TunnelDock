package tunnel

import "testing"

func TestFormatForwardSpec(t *testing.T) {
	tests := []struct {
		name         string
		localAddress string
		localPort    uint16
		remoteHost   string
		remotePort   uint16
		want         string
	}{
		{name: "hostname", localAddress: "localhost", localPort: 8888, remoteHost: "service.internal", remotePort: 8888, want: "localhost:8888:service.internal:8888"},
		{name: "IPv4", localAddress: "127.0.0.1", localPort: 18888, remoteHost: "192.168.10.50", remotePort: 8888, want: "127.0.0.1:18888:192.168.10.50:8888"},
		{name: "IPv6 local", localAddress: "::1", localPort: 8888, remoteHost: "127.0.0.1", remotePort: 8888, want: "[::1]:8888:127.0.0.1:8888"},
		{name: "IPv6 remote", localAddress: "127.0.0.1", localPort: 8888, remoteHost: "2001:db8::10", remotePort: 8888, want: "127.0.0.1:8888:[2001:db8::10]:8888"},
		{name: "both IPv6", localAddress: "[::1]", localPort: 8888, remoteHost: "[2001:db8::10]", remotePort: 8888, want: "[::1]:8888:[2001:db8::10]:8888"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatForwardSpec(tt.localAddress, tt.localPort, tt.remoteHost, tt.remotePort)
			if err != nil {
				t.Fatalf("FormatForwardSpec() error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("FormatForwardSpec() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatForwardSpecRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		local  string
		lp     uint16
		remote string
		rp     uint16
	}{
		{local: "", lp: 8888, remote: "host", rp: 8888},
		{local: "127.0.0.1", lp: 0, remote: "host", rp: 8888},
		{local: "127.0.0.1", lp: 8888, remote: "host\n-L", rp: 8888},
	} {
		if _, err := FormatForwardSpec(test.local, test.lp, test.remote, test.rp); err == nil {
			t.Fatalf("FormatForwardSpec(%q, %d, %q, %d) accepted invalid input", test.local, test.lp, test.remote, test.rp)
		}
	}
}
