package network

import (
	"testing"

	"github.com/moby/moby/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestNetworkServiceReferencesIncludesIdleServicesAndResolvedEndpoints(t *testing.T) {
	services := []swarm.Service{
		{ID: "idle", Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "zero-replicas"},
			Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: new(uint64(0))}},
			TaskTemplate: swarm.TaskSpec{Networks: []swarm.NetworkAttachmentConfig{{Target: "network-id"}}},
		}},
		{ID: "by-name", Spec: swarm.ServiceSpec{TaskTemplate: swarm.TaskSpec{Networks: []swarm.NetworkAttachmentConfig{{Target: "network-name"}}}}},
		{ID: "endpoint", Endpoint: swarm.Endpoint{VirtualIPs: []swarm.EndpointVirtualIP{{NetworkID: "network-id"}}}},
		{ID: "unrelated", Spec: swarm.ServiceSpec{TaskTemplate: swarm.TaskSpec{Networks: []swarm.NetworkAttachmentConfig{{Target: "other-id"}}}}},
		{ID: "no-networks"},
	}
	require.Equal(t, []string{"zero-replicas", "by-name", "endpoint"}, networkServiceReferences("network-id", "network-name", services))
	require.Empty(t, networkServiceReferences("absent", "", services))
}
