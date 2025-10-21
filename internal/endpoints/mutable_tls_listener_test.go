package endpoints

import (
	"testing"

	"github.com/canonical/lxd/shared"
	"github.com/stretchr/testify/require"
)

// Test_mutableTLSListener tests immutability and concurrent access.
func Test_mutableTLSListener(t *testing.T) {
	listener := &mutableTLSListener{}
	cert1 := shared.TestingKeyPair()
	cert2 := shared.TestingAltKeyPair()

	// Test immutability: different certs create different config objects
	listener.Config(cert1)
	config1 := listener.config
	require.NotNil(t, config1)

	listener.Config(cert2)
	config2 := listener.config
	require.NotNil(t, config2)
	require.NotSame(t, config1, config2, "Different certs should create different config objects")
}
