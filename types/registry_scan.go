package types

import "time"

type (
	RegistryScanReport struct {
		CompletedAt time.Time             `json:"completedAt,omitzero"`
		Digest      string                `json:"digest"`
		Status      string                `json:"status"`
		Severity    string                `json:"severity,omitempty"`
		Scanner     string                `json:"scanner,omitempty"`
		Platform    string                `json:"platform,omitempty"`
		Counts      map[string]int        `json:"counts,omitempty"`
		Findings    []RegistryScanFinding `json:"findings"`
		Total       int                   `json:"total"`
	}

	RegistryScanFinding struct {
		ID               string   `json:"id"`
		Package          string   `json:"package"`
		InstalledVersion string   `json:"version"`
		FixedVersion     string   `json:"fix_version,omitempty"`
		Severity         string   `json:"severity"`
		Links            []string `json:"links,omitempty"`
	}
)
