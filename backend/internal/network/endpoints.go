package network

import (
	"encoding/hex"

	"github.com/moby/moby/api/types/network"
)

// Network inspection also includes synthetic endpoints such as lb-<network>.
func networkContainerEndpoints(endpoints map[string]network.EndpointResource) map[string]network.EndpointResource {
	containers := make(map[string]network.EndpointResource)
	for id, endpoint := range endpoints {
		if len(id) != 64 {
			continue
		}
		if _, err := hex.DecodeString(id); err != nil {
			continue
		}
		containers[id] = endpoint
	}
	return containers
}
