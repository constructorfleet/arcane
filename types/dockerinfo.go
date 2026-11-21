package types

import (
	"github.com/moby/moby/api/types/system"
)

type (
	DockerInfo struct {
		APIVersion string `json:"apiVersion"`
		GitCommit  string `json:"gitCommit"`
		GoVersion  string `json:"goVersion"`
		Os         string `json:"os"`
		Arch       string `json:"arch"`
		BuildTime  string `json:"buildTime"`
		system.Info
		Success bool `json:"success"`
	}
	DockerPluginSummary struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Reference    string   `json:"reference,omitempty"`
		Capabilities []string `json:"capabilities"`
		Enabled      bool     `json:"enabled"`
	}
)
