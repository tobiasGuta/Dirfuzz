package campaign

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

type CampaignSnapshot struct {
	ID       string
	TargetID string
	Nodes    map[string]EvidenceProjection
}

type DiffEngine interface {
	Compare(snapOld CampaignSnapshot, snapNew CampaignSnapshot) CampaignDiff
}

type DefaultDiffEngine struct{}

func computeDiffHash(oldNode, newNode EvidenceProjection) string {
	// Hash the full transition rather than only the risk score: two independent
	// changes on the same path must not share a diff-memory fingerprint.
	pair := struct {
		Old EvidenceProjection `json:"old"`
		New EvidenceProjection `json:"new"`
	}{Old: oldNode, New: newNode}
	encoded, _ := json.Marshal(pair) // EvidenceProjection has only JSON-safe fields.
	return fmt.Sprintf("diff-%x", sha256.Sum256(encoded))
}

func (e *DefaultDiffEngine) Compare(snapOld CampaignSnapshot, snapNew CampaignSnapshot) CampaignDiff {
	diff := CampaignDiff{
		Endpoints: make([]EndpointDiff, 0),
	}

	oldNodes := make(map[string]EvidenceProjection, len(snapOld.Nodes))
	for k, v := range snapOld.Nodes {
		oldNodes[k] = v
	}

	newKeys := make([]string, 0, len(snapNew.Nodes))
	for k := range snapNew.Nodes {
		newKeys = append(newKeys, k)
	}
	sort.Strings(newKeys)

	for _, nodeID := range newKeys {
		newNode := snapNew.Nodes[nodeID]
		oldNode, exists := oldNodes[nodeID]
		if !exists {
			diff.TotalNew++
			diff.Endpoints = append(diff.Endpoints, EndpointDiff{
				NodeID:   nodeID,
				Path:     newNode.Path,
				Category: DiffNew,
				Severity: DiffInteresting,
				Fingerprint: DiffFingerprint{
					TargetID: snapNew.TargetID,
					DiffHash: fmt.Sprintf("new-%s", newNode.Path),
				},
				Current:  newNode,
			})
		} else {
			reasons, sev := detectChanges(oldNode, newNode)
			if len(reasons) > 0 {
				diff.TotalChange++
				diff.Endpoints = append(diff.Endpoints, EndpointDiff{
					NodeID:   nodeID,
					Path:     newNode.Path,
					Category: DiffChanged,
					Severity: sev,
					Fingerprint: DiffFingerprint{
						TargetID: snapNew.TargetID,
						DiffHash: computeDiffHash(oldNode, newNode),
					},
					Reasons:  reasons,
					Previous: oldNode,
					Current:  newNode,
				})
			}
			delete(oldNodes, nodeID)
		}
	}

	oldKeys := make([]string, 0, len(oldNodes))
	for k := range oldNodes {
		oldKeys = append(oldKeys, k)
	}
	sort.Strings(oldKeys)

	for _, nodeID := range oldKeys {
		oldNode := oldNodes[nodeID]
		diff.TotalRemove++
		diff.Endpoints = append(diff.Endpoints, EndpointDiff{
			NodeID:   nodeID,
			Path:     oldNode.Path,
			Category: DiffRemoved,
			Severity: DiffInfo,
			Fingerprint: DiffFingerprint{
				TargetID: snapOld.TargetID,
				DiffHash: fmt.Sprintf("rem-%s", oldNode.Path),
			},
			Previous: oldNode,
		})
	}

	return diff
}

func detectChanges(oldNode, newNode EvidenceProjection) ([]ChangeReason, DiffSeverity) {
	var reasons []ChangeReason
	severity := DiffInfo

	if oldNode.StatusCode != newNode.StatusCode {
		reasons = append(reasons, ChangeReason{
			Type: StatusChanged,
			Before: fmt.Sprintf("%d", oldNode.StatusCode),
			After: fmt.Sprintf("%d", newNode.StatusCode),
		})
		severity = DiffInteresting
		// A previously protected endpoint becoming successful is an auth
		// boundary regression worth escalating independently of any risk score.
		if (oldNode.StatusCode == 401 || oldNode.StatusCode == 403) &&
			newNode.StatusCode >= 200 && newNode.StatusCode < 300 {
			reasons = append(reasons, ChangeReason{
				Type: AuthChanged,
				Before: fmt.Sprintf("%d", oldNode.StatusCode),
				After: fmt.Sprintf("%d", newNode.StatusCode),
			})
			severity = DiffCritical
		}
	}
	if oldNode.Size != newNode.Size {
		reasons = append(reasons, ChangeReason{
			Type: SizeChanged,
			Before: fmt.Sprintf("%d", oldNode.Size),
			After: fmt.Sprintf("%d", newNode.Size),
		})
		if severity == DiffInfo {
			severity = DiffInteresting
		}
	}
	if oldNode.ContentType != newNode.ContentType {
		reasons = append(reasons, ChangeReason{
			Type: ContentTypeChanged,
			Before: oldNode.ContentType,
			After: newNode.ContentType,
		})
		if severity == DiffInfo {
			severity = DiffInteresting
		}
	}
	if oldNode.Hash != "" && newNode.Hash != "" && oldNode.Hash != newNode.Hash {
		reasons = append(reasons, ChangeReason{
			Type: BodyHashChanged,
			Before: oldNode.Hash,
			After: newNode.Hash,
		})
		if severity == DiffInfo {
			severity = DiffInteresting
		}
	}
	if oldNode.RiskScore != newNode.RiskScore {
		reasons = append(reasons, ChangeReason{
			Type: RiskChanged,
			Before: fmt.Sprintf("%d", oldNode.RiskScore),
			After: fmt.Sprintf("%d", newNode.RiskScore),
		})
		diffAmt := newNode.RiskScore - oldNode.RiskScore
		if diffAmt < 0 {
			diffAmt = -diffAmt
		}
		if diffAmt > 50 {
			severity = DiffCritical
		} else if severity == DiffInfo {
			severity = DiffInteresting
		}
	}

	return reasons, severity
}
