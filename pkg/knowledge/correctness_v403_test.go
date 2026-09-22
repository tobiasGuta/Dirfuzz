package knowledge

import (
	"testing"
	"time"
)

func TestCampaignScopedDecisionIsStoredAndIsolated(t *testing.T) {
	store := NewMemoryStore()
	sig := PatternSignature{Method: "GET", RouteClass: "/admin", StatusCode: 403}
	ctxA := KnowledgeContext{TargetID: "target-1", ProjectID: "project-1", CampaignID: "campaign-a"}
	ctxB := KnowledgeContext{TargetID: "target-1", ProjectID: "project-1", CampaignID: "campaign-b"}

	if err := store.RecordDecision(AnalystDecision{
		ID: "confirmed-a", Decision: DecisionConfirmed,
		Metadata: DecisionMetadata{Confidence: 2, EvidenceCount: 3},
	}, sig, ctxA, ScopeCampaign); err != nil {
		t.Fatal(err)
	}
	if got := store.LookupPattern(sig, ctxA, time.Now()); got.ConfidenceBoost == 0 || got.ConfidencePenalty != 0 {
		t.Fatalf("same-campaign confirmation should boost confidence: %+v", got)
	}
	if got := store.LookupPattern(sig, ctxB, time.Now()); got.ConfidenceBoost != 0 || got.ConfidencePenalty != 0 {
		t.Fatalf("campaign-a evidence leaked into campaign-b: %+v", got)
	}
}

func TestRejectedDecisionNeverBoostsConfidence(t *testing.T) {
	store := NewMemoryStore()
	sig := PatternSignature{Method: "GET", RouteClass: "/noisy", StatusCode: 200}
	ctx := KnowledgeContext{TargetID: "target-1"}
	if err := store.RecordDecision(AnalystDecision{ID: "rejected-1", Decision: DecisionRejected}, sig, ctx, ScopeTarget); err != nil {
		t.Fatal(err)
	}
	got := store.LookupPattern(sig, ctx, time.Now())
	if got.ConfidencePenalty <= 0 || got.ConfidenceBoost != 0 {
		t.Fatalf("rejected decision must only penalize: %+v", got)
	}
	other := store.LookupPattern(sig, KnowledgeContext{TargetID: "target-2"}, time.Now())
	if other.ConfidencePenalty != 0 || other.ConfidenceBoost != 0 {
		t.Fatalf("target-scoped rejection leaked into different target: %+v", other)
	}
}
