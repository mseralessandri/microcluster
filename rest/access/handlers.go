package access

import (
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/lxd/shared/logger"

	"github.com/canonical/microcluster/v3/internal/endpoints"
	"github.com/canonical/microcluster/v3/internal/rest/access"
	"github.com/canonical/microcluster/v3/internal/rest/client"
	internalState "github.com/canonical/microcluster/v3/internal/state"
	"github.com/canonical/microcluster/v3/rest/types"
	"github.com/canonical/microcluster/v3/state"
)

// ErrInvalidHost is used to indicate that a request host is invalid.
type ErrInvalidHost struct {
	error
}

// Unwrap implements xerrors.Unwrap for ErrInvalidHost.
func (e ErrInvalidHost) Unwrap() error {
	return e.error
}

// AllowAuthenticated checks if the request is trusted by extracting access.TrustedRequest from the request context.
// This handler is used as an access handler by default if AllowUntrusted is false on a rest.EndpointAction.
func AllowAuthenticated(state state.State, r *http.Request) (bool, response.Response) {
	trusted := r.Context().Value(client.CtxAccess)
	if trusted == nil {
		return false, response.Forbidden(nil)
	}

	trustedReq, ok := trusted.(access.TrustedRequest)
	if !ok {
		return false, response.Forbidden(nil)
	}

	if !trustedReq.Trusted {
		return false, response.Forbidden(nil)
	}

	return true, nil
}

// Authenticate ensures the request certificates are trusted against the given set of trusted certificates.
// - Requests over the unix socket are always allowed.
// - HTTP requests require the TLS Peer certificate to match an entry in the supplied map of certificates.
func Authenticate(state state.State, r *http.Request, hostAddress string, trustedCerts map[string]x509.Certificate) (bool, error) {
	if r.RemoteAddr == "@" {
		return true, nil
	}

	intState, err := internalState.ToInternal(state)
	if err != nil {
		return false, err
	}

	// Check if it's the core API listener and if it is using the server.crt.
	// This indicates that the daemon is in a pre-init state and is listening on the PreInitListenAddress.
	endpoint := intState.Endpoints.Get(endpoints.EndpointsCore)
	network, ok := endpoint.(*endpoints.Network)
	if ok {
		if state.ServerCert().Fingerprint() == network.TLS().Fingerprint() {
			logger.Info("Allowing unauthenticated request to un-initialized system")
			return true, nil
		}
	}

	// Ensure the given host address is valid.
	hostAddrPort, err := types.ParseAddrPort(hostAddress)
	if err != nil {
		return false, fmt.Errorf("Invalid host address %q", hostAddress)
	}

	switch r.Host {
	case hostAddrPort.WithZone("").String():
		if r.TLS != nil {
			for _, cert := range r.TLS.PeerCertificates {
				trusted, fingerprint := util.CheckMutualTLS(*cert, trustedCerts)
				if trusted {
					logger.Debugf("Trusting HTTP request to %q from %q with fingerprint %q", r.URL.String(), r.RemoteAddr, fingerprint)

					return trusted, nil
				}
			}
		}

	default:
		return false, ErrInvalidHost{error: fmt.Errorf("Invalid request address %q", r.Host)}
	}

	return false, nil
}
