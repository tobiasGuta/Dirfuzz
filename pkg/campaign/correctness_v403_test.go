package campaign

import "testing"

func TestV403StructuralChangesWithStableRiskScore(t *testing.T) {
	base := EvidenceProjection{
		Path: "/api/account", StatusCode: 403,
		ContentType: "application/json", Size: 100, Hash: "before", RiskScore: 20,
	}
	tests := []struct {
		name string
		mutate func(*EvidenceProjection)
		wantReason ChangeReasonType
		wantSeverity DiffSeverity
	}{
		{"auth boundary 403 to 200", func(p *EvidenceProjection) { p.StatusCode = 200 }, AuthChanged, DiffCritical},
		{"size only", func(p *EvidenceProjection) { p.Size = 101 }, SizeChanged, DiffInteresting},
		{"content type only", func(p *EvidenceProjection) { p.ContentType = "text/html" }, ContentTypeChanged, DiffInteresting},
		{"body hash only", func(p *EvidenceProjection) { p.Hash = "after" }, BodyHashChanged, DiffInteresting},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := base
			tc.mutate(&next)
			reasons, sev := detectChanges(base, next)
			if sev != tc.wantSeverity {
				t.Fatalf("severity = %s, want %s", sev, tc.wantSeverity)
			}
			found := false
			for _, reason := range reasons {
				if reason.Type == tc.wantReason { found = true }
			}
			if !found { t.Fatalf("missing %s in reasons: %+v", tc.wantReason, reasons) }
			oldSnap := CampaignSnapshot{TargetID: "target-1", Nodes: map[string]EvidenceProjection{"endpoint-1": base}}
			newSnap := CampaignSnapshot{TargetID: "target-1", Nodes: map[string]EvidenceProjection{"endpoint-1": next}}
			diff := (&DefaultDiffEngine{}).Compare(oldSnap, newSnap)
			if diff.TotalChange != 1 || len(diff.Endpoints) != 1 {
				t.Fatalf("structural change was not recorded: %+v", diff)
			}
		})
	}
}

func TestV403UnchangedEvidenceDoesNotProduceDiff(t *testing.T) {
	projection := EvidenceProjection{
		Path: "/api/account", StatusCode: 200, ContentType: "application/json",
		Size: 123, Hash: "body-hash", RiskScore: 20,
	}
	reasons, severity := detectChanges(projection, projection)
	if len(reasons) != 0 || severity != DiffInfo {
		t.Fatalf("unchanged evidence produced a diff: reasons=%+v severity=%s", reasons, severity)
	}
}

func TestV403DiffFingerprintIncludesStructuralTransition(t *testing.T) {
	base := EvidenceProjection{Path: "/api/account", StatusCode: 403, RiskScore: 20}
	statusChanged := base
	statusChanged.StatusCode = 200
	sizeChanged := base
	sizeChanged.Size = 123
	a := computeDiffHash(base, statusChanged)
	b := computeDiffHash(base, sizeChanged)
	if a == b {
		t.Fatalf("distinct transitions share diff fingerprint %q", a)
	}
	if a != computeDiffHash(base, statusChanged) {
		t.Fatalf("same transition produced nondeterministic fingerprint")
	}
}
