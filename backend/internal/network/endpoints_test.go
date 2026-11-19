package network

import (
	"strings"
	"testing"

	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
)

func TestNetworkContainerEndpointsPreservesRealConsumers(t *testing.T) {
	applicationID := strings.Repeat("a", 64)
	helperID := strings.Repeat("b", 64)
	endpoints := map[string]network.EndpointResource{
		applicationID:           {Name: "lb-application"},
		helperID:                {Name: "arcane-volume-helper"},
		"lb-traefik-public":     {Name: "traefik-public-endpoint"},
		"ingress-sbox":          {Name: "ingress-endpoint"},
		strings.Repeat("z", 64): {Name: "invalid-id"},
	}
	require.Equal(t, map[string]network.EndpointResource{
		applicationID: {Name: "lb-application"},
		helperID:      {Name: "arcane-volume-helper"},
	}, networkContainerEndpoints(endpoints))
	require.Empty(t, networkContainerEndpoints(map[string]network.EndpointResource{"lb-unused": {Name: "unused-endpoint"}}))
}
