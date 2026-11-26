package browse

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

type (
	harborScanArtifact struct {
		ExtraAttrs   harborScanPlatform            `json:"extra_attrs"`
		Digest       string                        `json:"digest"`
		Overview     map[string]harborScanOverview `json:"scan_overview"`
		ManifestType string                        `json:"manifest_media_type"`
		References   []harborScanReference         `json:"references"`
	}

	harborScanReference struct {
		Digest   string             `json:"child_digest"`
		Platform harborScanPlatform `json:"platform"`
	}

	harborScanPlatform struct {
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
		Variant      string `json:"variant"`
	}

	harborScanOverview struct {
		CompletedAt time.Time `json:"end_time"`
		Status      string    `json:"scan_status"`
		Severity    string    `json:"severity"`
		Scanner     struct {
			Name string `json:"name"`
		} `json:"scanner"`
		Summary struct {
			Counts map[string]int `json:"summary"`
			Total  int            `json:"total"`
		} `json:"summary"`
	}

	harborScanReport struct {
		Findings []types.RegistryScanFinding `json:"vulnerabilities"`
	}
)

const (
	harborVulnerabilityMediaType = "application/vnd.security.vulnerability.report; version=1.1"
	maxHarborScanResponseBytes   = 24 << 20
)

func (p harborScanPlatform) matches(platform string) bool {
	base := p.OS + "/" + p.Architecture
	return platform == base || (p.Variant != "" && platform == base+"/"+p.Variant)
}

func (s *Service) ScanReport(ctx context.Context, id, repository, reference, platform string) (types.RegistryScanReport, error) {
	ctx, cancel := s.lookupContext(ctx)
	defer cancel()
	target, err := s.target(ctx, id)
	if err != nil {
		return types.RegistryScanReport{}, err
	}
	repo, err := target.repository(repository)
	if err != nil {
		return types.RegistryScanReport{}, err
	}
	project, name, ok := strings.Cut(repo.RepositoryStr(), "/")
	if !ok {
		return types.RegistryScanReport{}, common.Classify(common.ErrValidation, errors.New("a Harbor project and repository are required"))
	}
	if _, parseErr := digest.Parse(reference); parseErr != nil {
		return types.RegistryScanReport{}, common.Classify(common.ErrValidation, errors.New("a valid manifest digest is required"))
	}
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
			return types.RegistryScanReport{}, common.Classify(common.ErrValidation, errors.New("invalid image platform"))
		}
	}
	scheme, err := s.harborScheme(ctx, target)
	if err != nil {
		return types.RegistryScanReport{}, classifyBrowseError(err, "failed to connect to registry")
	}
	// Harbor requires the repository segment to be escaped twice when it contains slashes.
	path := "/api/v2.0/projects/" + project + "/repositories/" + url.PathEscape(name) + "/artifacts/"
	artifact, err := s.readHarborScanArtifact(ctx, target, scheme, path, reference)
	if err != nil {
		return types.RegistryScanReport{}, err
	}
	if platform != "" && (len(artifact.References) > 0 || strings.Contains(artifact.ManifestType, "image.index") || strings.Contains(artifact.ManifestType, "manifest.list")) {
		var matching []string
		for _, child := range artifact.References {
			if child.Platform.matches(platform) {
				matching = append(matching, child.Digest)
			}
		}
		if len(matching) != 1 {
			return types.RegistryScanReport{Digest: reference, Platform: platform, Status: "Platform unavailable"}, nil
		}
		artifact, err = s.readHarborScanArtifact(ctx, target, scheme, path, matching[0])
		if err != nil {
			return types.RegistryScanReport{}, err
		}
	}

	if platform != "" && !artifact.ExtraAttrs.matches(platform) {
		return types.RegistryScanReport{Digest: reference, Platform: platform, Status: "Platform unavailable"}, nil
	}

	overview, found := artifact.Overview[harborVulnerabilityMediaType]
	if !found {
		return types.RegistryScanReport{Digest: artifact.Digest, Platform: platform, Status: "Not scanned"}, nil
	}
	result := types.RegistryScanReport{
		Digest: artifact.Digest, Platform: platform, Status: overview.Status, Severity: overview.Severity,
		CompletedAt: overview.CompletedAt, Scanner: overview.Scanner.Name, Total: overview.Summary.Total, Counts: overview.Summary.Counts,
	}
	if overview.Status != "Success" {
		return result, nil
	}
	body, err := s.getHarborScanData(ctx, target, scheme, path+artifact.Digest+"/additions/vulnerabilities", nil)
	if err != nil {
		return types.RegistryScanReport{}, err
	}
	var reports map[string]harborScanReport
	if decodeErr := json.Unmarshal(body, &reports); decodeErr != nil {
		return types.RegistryScanReport{}, fmt.Errorf("decode Harbor scan report: %w", decodeErr)
	}
	report, found := reports[harborVulnerabilityMediaType]
	if !found || report.Findings == nil {
		return types.RegistryScanReport{}, errors.New("harbor did not return the requested vulnerability report format")
	}
	result.Findings = report.Findings
	return result, nil
}

func (s *Service) readHarborScanArtifact(ctx context.Context, target *target, scheme, path, reference string) (harborScanArtifact, error) {
	if _, err := digest.Parse(reference); err != nil {
		return harborScanArtifact{}, errors.New("harbor returned an invalid platform digest")
	}
	body, err := s.getHarborScanData(ctx, target, scheme, path+reference, url.Values{"with_scan_overview": {"true"}})
	if err != nil {
		return harborScanArtifact{}, err
	}
	var artifact harborScanArtifact
	if decodeErr := json.Unmarshal(body, &artifact); decodeErr != nil {
		return harborScanArtifact{}, fmt.Errorf("decode Harbor artifact: %w", decodeErr)
	}
	if artifact.Digest != reference {
		return harborScanArtifact{}, errors.New("harbor returned a different artifact digest")
	}
	return artifact, nil
}

func (s *Service) getHarborScanData(ctx context.Context, target *target, scheme, path string, query url.Values) ([]byte, error) {
	response, err := s.harborGET(ctx, target, scheme, path, query, harborVulnerabilityMediaType)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, common.Classify(common.ErrForbidden, errors.New("harbor denied access to the vulnerability data"))
	case http.StatusNotFound:
		return nil, common.Classify(common.ErrNotFound, errors.New("artifact or scan report was not found in Harbor"))
	default:
		return nil, fmt.Errorf("harbor vulnerability API returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHarborScanResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxHarborScanResponseBytes {
		return nil, errors.New("harbor vulnerability response is too large")
	}
	return body, nil
}
