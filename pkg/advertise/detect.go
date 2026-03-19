package advertise

import (
	"fmt"
	"net"
	"os"
)

// isKubernetes returns true if running inside a Kubernetes pod.
func isKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// detectOutboundIP discovers the primary outbound interface IP by opening
// a UDP socket to a well-known address. No traffic is sent.
func detectOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp4", "1.1.1.1:53")
	if err != nil {
		return nil, fmt.Errorf("detect outbound IP: %w", err)
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local address type: %T", conn.LocalAddr())
	}

	if addr.IP.IsLoopback() || addr.IP.IsUnspecified() {
		return nil, fmt.Errorf("detected loopback/unspecified IP %s", addr.IP)
	}

	return addr.IP, nil
}
