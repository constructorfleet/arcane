package types

import "time"

type (
	RegistryArtifactScanOptions struct {
		RegistryID string
		Repository string
		Digest     string
		Platform   string
		Trigger    bool
	}
	ImageRegistryScanOptions struct {
		Platform   string
		References []string
		Trigger    bool
	}

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

	RegistryScanCVSS struct {
		V3Vector string  `json:"vector_v3"`
		V2Vector string  `json:"vector_v2"`
		V3Score  float64 `json:"score_v3"`
		V2Score  float64 `json:"score_v2"`
	}

	RegistryScanFinding struct {
		CVSS             *RegistryScanCVSS `json:"preferred_cvss,omitempty"`
		Description      string            `json:"description,omitempty"`
		ID               string            `json:"id"`
		Package          string            `json:"package"`
		InstalledVersion string            `json:"version"`
		FixedVersion     string            `json:"fix_version,omitempty"`
		Severity         string            `json:"severity"`
		Links            []string          `json:"links,omitempty"`
	}
)
