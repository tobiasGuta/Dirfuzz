# Architecture Freeze

The intelligence framework of DirFuzz is **frozen**. The project has evolved from a directory fuzzer into a Bug Bounty Operations Platform. To ensure the platform survives operational reality without succumbing to feature bloat, the following laws govern all future development:

## 1. No New Intelligence Layers
Algorithms are no longer the bottleneck. The focus is exclusively on Deployment, Storage, Connectors, and Data Quality. Do not propose new AI models, ranking heuristics, or analytical projections until the existing ones have failed under the weight of real-world noise.

## 2. No Direct Mutation of Projections
Projections (`CampaignGraph`, `DiscoveryGraph`, `KnowledgeStore`) are strictly read-only materializations of the Ledger.
- Subsystems may never directly edit another subsystem's state.
- All state changes flow entirely through immutable Ledger Events.

## 3. No Autonomous Validation
The system may act as an intelligence brain and suggest actions (`PlaybookSuggestion`), but it is explicitly forbidden from silently executing automated loops without Analyst Validation. Humans remain firmly in the loop.

## 4. The Event Ledger Remains Absolute Truth
If an action, discovery, or analyst decision is not appended to the Event Ledger, it never happened. History is append-only. 

## 5. Analyst Workload Reduction
New features must measurably reduce the Analyst's cognitive load. Do not add metrics that cannot be actioned. Do not add alerts that lack structural context (`ExecutionID`, `Asset`, `Reason`).
