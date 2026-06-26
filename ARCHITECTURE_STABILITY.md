# Architecture Stability Rules

DirFuzz has formalized its architecture as a **Bug Bounty Intelligence OS**. To prevent future feature drift and "quick PRs" from degrading the platform back into a standard scanner, all new development must adhere to the following invariant rules.

## 1. The Event Ledger is the Single Source of Truth
No subsystem is permitted to track its own state natively. Every state change (Discoveries, Analyst Decisions, Diffs, Playbook execution) MUST be written to the Ledger as an immutable `Event`.
- **Rule**: If an event is not in the ledger, it never happened.
- **Rule**: The `EventStore` is purely a historical storage contract (`Append`, `Read`, `Replay`). It MUST NEVER contain business logic or domain queries (e.g., `store.FindCriticalEndpoints()`). Querying is strictly the responsibility of the Projections built after `Read()`.

## 2. All State is a Deterministic Projection
Subsystems (`DiscoveryGraph`, `CampaignGraph`, `KnowledgeStore`, `RiskAnalytics`) are strictly **Projections**. They reconstruct their state by chronologically reading the Event Ledger.
- **Rule**: Replaying the exact same Ledger MUST yield the exact same Projection, pixel-for-pixel and byte-for-byte. `Replay(Ledger) == Same Projection` every single time.
- **Rule**: Projections may safely be deleted, wiped from memory, or crashed without losing any data. 

## 3. Subsystem Isolation
No projection may mutate the state of another projection directly.
- **Rule**: The UI cannot directly edit the `DiscoveryGraph`. The UI creates an `AnalystDecisionEvent` in the Ledger. The Projection Engine reads the event and recalculates the graph. 
- **Rule**: `Campaign A` cannot read or mutate edges belonging to `Campaign B`. (Tested via `TestCampaignIsolation`).

## 4. Evidence Immutability (Append-Only)
Events can never be edited or deleted.
- **Rule**: If a `403` changes to a `200`, you do NOT rewrite the past `403` event. You append a new `RegressionEvent`. The Projection calculates the diff. (Tested via `TestEvidenceCannotRewriteHistory`).

## 5. Human-in-the-loop Validation
The engine does not have autonomy.
- **Rule**: Intelligence layers (`Playbooks`) generate `PlaybookSuggestion` events. They cannot execute scanners. Only the receipt of an `AnalystValidationCommand` from a human (or a confirmed automation policy via the TUI) is permitted to kick off new Engine Work.
