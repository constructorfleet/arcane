package network

import (
	"cmp"

	"github.com/moby/moby/api/types/swarm"
)

func networkServiceReferences(id, name string, services []swarm.Service) []string {
	var references []string
	for _, service := range services {
		referenced := false
		for _, attachment := range service.Spec.TaskTemplate.Networks {
			if attachment.Target == id || (name != "" && attachment.Target == name) {
				referenced = true
				break
			}
		}
		if !referenced {
			for _, address := range service.Endpoint.VirtualIPs {
				if address.NetworkID == id {
					referenced = true
					break
				}
			}
		}
		if referenced {
			references = append(references, cmp.Or(service.Spec.Name, service.ID))
		}
	}
	return references
}
