package upgrade

import (
	"errors"
	"fmt"
	"strings"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func selfUpgradeRestriction(labels map[string]string) error {
	if strings.TrimSpace(labels["com.docker.swarm.service.id"]) != "" || strings.TrimSpace(labels["com.docker.swarm.service.name"]) != "" {
		return common.Classify(common.ErrConflict, errors.New("swarm task updates are managed through the service or stack deployment configuration"))
	}
	if unit := types.QuadletUnit(labels); unit != "" {
		return fmt.Errorf("systemd unit %s: %w", unit, common.ErrContainerQuadletManaged)
	}
	return nil
}
