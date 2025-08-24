package tcp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type TCPProbe struct {
	Addresses []string
	Timeout   time.Duration
	Region    string
	// No connection reuse; stateless
}

func NewTCPProbe(addresses []string, timeout time.Duration) *TCPProbe {
	return &TCPProbe{
		Addresses: addresses,
		Timeout:   timeout,
		Region:    "",
	}
}

// Probe implements the Prober interface
func (p *TCPProbe) Probe(ctx context.Context) error {
	var errs []string
	for _, addr := range p.Addresses {
		conn, err := net.DialTimeout("tcp", addr, p.Timeout)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: dial error: %v", addr, err))
			continue
		}
		// Set a deadline for liveness check
		if err := conn.SetDeadline(time.Now().Add(p.Timeout)); err != nil {
			errs = append(errs, fmt.Sprintf("%s: set deadline error: %v", addr, err))
			conn.Close()
			continue
		}
		_, err = conn.Write([]byte{})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: write error: %v", addr, err))
			conn.Close()
			continue
		}
		conn.Close()
	}
	if len(errs) > 0 {
		return fmt.Errorf("tcp probe errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (p *TCPProbe) Close() {}

func (p *TCPProbe) MetadataString() string {
	return fmt.Sprintf("addresses: %v | region: %s", p.Addresses, p.Region)
}
