package engine

import (
	"hash/fnv"
	"math/bits"
	"sync"
	"unicode"
	"unicode/utf8"
)

// simhashBody computes a 64-bit SimHash fingerprint for a response body without allocating.
func simhashBody(body []byte) uint64 {
	if len(body) == 0 {
		return 0
	}

	var vector [64]int
	hasTokens := false

	inToken := false
	tokenStart := 0

	for i := 0; i < len(body); {
		r, size := utf8.DecodeRune(body[i:])
		isBoundary := unicode.IsSpace(r) || unicode.IsPunct(r)

		if isBoundary {
			if inToken {
				token := body[tokenStart:i]
				if len(token) > 0 {
					hashToken(token, &vector)
					hasTokens = true
				}
				inToken = false
			}
		} else {
			if !inToken {
				tokenStart = i
				inToken = true
			}
		}
		i += size
	}

	if inToken {
		token := body[tokenStart:]
		if len(token) > 0 {
			hashToken(token, &vector)
			hasTokens = true
		}
	}

	if !hasTokens {
		return 0
	}

	var fingerprint uint64
	for bit, weight := range vector {
		if weight > 0 {
			fingerprint |= uint64(1) << bit
		}
	}
	return fingerprint
}

func hashToken(token []byte, vector *[64]int) {
	hasher := fnv.New64a()
	_, _ = hasher.Write(token)
	h := hasher.Sum64()

	for bit := 0; bit < 64; bit++ {
		if h&(uint64(1)<<bit) != 0 {
			vector[bit]++
		} else {
			vector[bit]--
		}
	}
}

func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// SimhashTracker manages SimHash-based soft-404 clustering.
type SimhashTracker struct {
	clusters     map[uint64]int
	clusterLock  sync.Mutex
	Threshold    int
	ClusterLimit int
}

// NewSimhashTracker creates a new SimhashTracker.
func NewSimhashTracker(threshold, limit int) *SimhashTracker {
	return &SimhashTracker{
		clusters:     make(map[uint64]int),
		Threshold:    threshold,
		ClusterLimit: limit,
	}
}

// Clear resets the cluster map.
func (s *SimhashTracker) Clear() {
	s.clusterLock.Lock()
	s.clusters = make(map[uint64]int)
	s.clusterLock.Unlock()
}

// IsSoftFour tracks a SimHash cluster and returns true once the cluster
// has reached the suppression limit.
func (s *SimhashTracker) IsSoftFour(bodyHash uint64) bool {
	threshold := s.Threshold
	if threshold < 0 {
		threshold = 0
	}
	limit := s.ClusterLimit
	if limit <= 0 {
		return false
	}

	s.clusterLock.Lock()
	defer s.clusterLock.Unlock()

	for centroid, count := range s.clusters {
		if hammingDistance(centroid, bodyHash) <= threshold {
			count++
			s.clusters[centroid] = count
			return count >= limit
		}
	}

	const maxSimhashCentroids = 5000
	if len(s.clusters) >= maxSimhashCentroids {
		var lowestCentroid uint64
		lowestCount := int(1e9)
		for centroid, count := range s.clusters {
			if count < lowestCount {
				lowestCount = count
				lowestCentroid = centroid
			}
		}
		delete(s.clusters, lowestCentroid)
	}

	s.clusters[bodyHash] = 1
	return false
}
