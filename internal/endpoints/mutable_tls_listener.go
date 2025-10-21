package endpoints

import (
	"crypto/tls"
	"net"
	"sync"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/lxd/shared"
)

// mutableTLSListener is a variation of the standard tls.Listener that supports
// atomically swapping the underlying TLS configuration.
// Requests served before the swap will continue using the old configuration.
// This implementation matches LXD's FancyTLSListener but excludes:
//   - trustedProxy []net.IP field (proxy protocol support not needed)
//   - TrustedProxy() method for setting proxy IPs
//   - isProxy() function and proxy protocol wrapping logic
//   - proxyproto.NewConn() calls in Accept()
type mutableTLSListener struct {
	net.Listener
	mu     sync.RWMutex
	config *tls.Config
}

// newMutableTLSListener creates a new mutableTLSListener.
func newMutableTLSListener(inner net.Listener, cert *shared.CertInfo) *mutableTLSListener {
	listener := &mutableTLSListener{
		Listener: inner,
	}

	listener.Config(cert)
	return listener
}

// Accept waits for and returns the next incoming TLS connection then use the
// current TLS configuration to handle it.
// EXCLUDED from LXD: proxy protocol check and wrapping:
//   - if isProxy(c.RemoteAddr().String(), l.trustedProxy) { c = proxyproto.NewConn(c, 0) }
func (l *mutableTLSListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	config := l.config

	return tls.Server(c, config), nil
}

// Config safely swaps the underlying TLS configuration.
func (l *mutableTLSListener) Config(cert *shared.CertInfo) {
	config := util.ServerTLSConfig(cert)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.config = config
}
