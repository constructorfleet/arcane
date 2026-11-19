package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"

	"kit.getarcane.app/updater"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func TestSelfUpgradeRestriction(t *testing.T) {
	for _, labels := range []map[string]string{{"com.docker.swarm.service.id": "service-123"}, {"com.docker.swarm.service.name": "arcane_manager"}} {
		err := selfUpgradeRestriction(labels)
		require.ErrorIs(t, err, common.ErrConflict)
		require.ErrorContains(t, err, "deployment configuration")
	}
	require.NoError(t, selfUpgradeRestriction(nil))
	require.NoError(t, selfUpgradeRestriction(map[string]string{"com.docker.compose.project": "arcane"}))
}

func TestControlPlaneUpgradePolicy(t *testing.T) {
	svc := NewService(ServiceParams{ControlPlane: true})
	canUpgrade, err := svc.CanUpgrade(t.Context())
	require.False(t, canUpgrade)
	require.ErrorIs(t, err, common.ErrConflict)
	_, err = svc.prepareUpgrade(t.Context(), types.Actor{}, updater.SelfUpdateTarget{}, "")
	require.ErrorContains(t, err, "deployment configuration")
	_, err = svc.StartUpgradeJob(t.Context(), types.Actor{}, nil)
	require.ErrorContains(t, err, "deployment configuration")
	require.False(t, svc.upgrading.Load())
	require.False(t, svc.startingJob.Load())
}
