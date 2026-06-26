import re
import sys

def process():
    path = r"d:\Tools\DirFuzz-Mcp-Monitor\pkg\engine\harvest.go"
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()

    # 1. Update queueResponseHarvestFromResult
    content = content.replace(
        "e.SubmitHarvestPath(normalized, job.RunID, job.HarvestDepth+1)",
        'e.SubmitHarvestPath(normalized, job.RunID, job.HarvestDepth+1, job.DiscoveryNodeID, "response", DiscoveryEvidence{Type: "response html/json extraction"})'
    )

    # 2. Update harvestEndpointsWithOptions

    # Redefine `add` func
    old_add = """	add := func(candidate string) string {
		normalized := canonicalHarvestCandidate(base, candidate)
		if normalized == "" {
			return ""
		}
		if _, exists := discovered[normalized]; !exists && e != nil {
			e.emitLogEvent(LogLevelInfo, LogCategoryDiscovery, EventHarvestDiscovery, fmt.Sprintf("discovered endpoint %s", normalized), map[string]interface{}{
				"path": normalized,
			})
		}
		discovered[normalized] = struct{}{}
		return normalized
	}"""
    
    new_add = """	var runID int64
	if e != nil {
		runID = atomic.LoadInt64(&e.RunID)
	}
	add := func(candidate string, parentID, sourceType string, evidence DiscoveryEvidence) string {
		normalized := canonicalHarvestCandidate(base, candidate)
		if normalized == "" {
			return ""
		}
		if _, exists := discovered[normalized]; !exists && e != nil {
			e.emitLogEvent(LogLevelInfo, LogCategoryDiscovery, EventHarvestDiscovery, fmt.Sprintf("discovered endpoint %s", normalized), map[string]interface{}{
				"path": normalized,
			})
			e.SubmitHarvestPath(normalized, runID, 1, parentID, sourceType, evidence)
		}
		discovered[normalized] = struct{}{}
		return normalized
	}"""
    content = content.replace(old_add, new_add)

    # Need to update calls to `add(candidate)`
    # 2a. Response root
    # 2b. JS
    # 2c. Response queues
    # 2d. OpenAPI
    # 2e. GraphQL

    # Since it's easier, we'll replace the block step-by-step
    
    # Root response
    content = content.replace(
        """				for _, candidate := range responseHarvestCandidatesForTarget(baseURL, rootBody, contentType) {
					if normalized := add(candidate); normalized != "" {""",
        """				rootNodeID := ""
				if e != nil && e.DiscoveryGraph != nil {
					rootNodeID = e.DiscoveryGraph.AddSourceNode(baseURL, "response")
				}
				for _, candidate := range responseHarvestCandidatesForTarget(baseURL, rootBody, contentType) {
					if normalized := add(candidate, rootNodeID, "response", DiscoveryEvidence{Type: "response html/json extraction", Source: baseURL}); normalized != "" {"""
    )

    # JS
    content = content.replace(
        """				for _, scriptURL := range scriptURLs {
					if scriptBody, _, err := fetchHarvestBody(ctx, client, scriptURL, nil); err == nil {
						for _, candidate := range extractJSHarvestCandidates(scriptBody) {
							add(candidate)
						}
					}
				}""",
        """				for _, scriptURL := range scriptURLs {
					if scriptBody, _, err := fetchHarvestBody(ctx, client, scriptURL, nil); err == nil {
						scriptNodeID := ""
						if e != nil && e.DiscoveryGraph != nil {
							scriptNodeID = e.DiscoveryGraph.AddSourceNode(scriptURL, "javascript")
						}
						for _, candidate := range extractJSHarvestCandidates(scriptBody) {
							add(candidate, scriptNodeID, "javascript", DiscoveryEvidence{Type: "regex endpoint extraction", Source: scriptURL})
						}
					}
				}"""
    )

    # Queue response
    content = content.replace(
        """			candidates := responseHarvestCandidatesForTarget(target.url, body, contentType)
			for _, candidate := range candidates {
				if normalized := add(candidate); normalized != "" && target.depth < opts.responseMaxDepth {""",
        """			candidates := responseHarvestCandidatesForTarget(target.url, body, contentType)
			targetNodeID := ""
			if e != nil && e.DiscoveryGraph != nil {
				targetNodeID = e.DiscoveryGraph.AddSourceNode(target.url, "response")
			}
			for _, candidate := range candidates {
				if normalized := add(candidate, targetNodeID, "response", DiscoveryEvidence{Type: "response html/json extraction", Source: target.url}); normalized != "" && target.depth < opts.responseMaxDepth {"""
    )

    # OpenAPI
    content = content.replace(
        """			for _, candidate := range extractOpenAPIHarvestCandidates(e, body) {
				add(candidate)
			}""",
        """			apiNodeID := ""
			if e != nil && e.DiscoveryGraph != nil {
				apiNodeID = e.DiscoveryGraph.AddSourceNode(path, "openapi")
			}
			for _, candidate := range extractOpenAPIHarvestCandidates(e, body) {
				add(candidate, apiNodeID, "openapi", DiscoveryEvidence{Type: "schema introspection", Source: path})
			}"""
    )

    # GraphQL
    content = content.replace(
        """			for _, candidate := range extractGraphQLHarvestCandidates(e, body) {
				add(candidate)
			}""",
        """			gqlNodeID := ""
			if e != nil && e.DiscoveryGraph != nil {
				gqlNodeID = e.DiscoveryGraph.AddSourceNode(path, "graphql")
			}
			for _, candidate := range extractGraphQLHarvestCandidates(e, body) {
				add(candidate, gqlNodeID, "graphql", DiscoveryEvidence{Type: "schema introspection", Source: path})
			}"""
    )
    
    # We might need to add `sync/atomic` if not imported
    if '"sync/atomic"' not in content:
        content = content.replace('"sync"', '"sync"\n\t"sync/atomic"')

    with open(path, "w", encoding="utf-8") as f:
        f.write(content)

process()
