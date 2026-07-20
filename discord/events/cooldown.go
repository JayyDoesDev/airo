package events

import (
	"sync"
	"time"
)

const userCooldown = 5 * time.Second
const maxDMsPerRequest = 1
const globalDMCooldown = 10 * time.Second
const globalSearchLimit = 10
const globalSearchWindow = time.Minute

var dmAllowlist = map[string]bool{
	"419958345487745035": true,
}

var (
	cooldowns    = map[string]time.Time{}
	cooldownsMu  sync.Mutex
	lastDMTime   time.Time
	dmCooldownMu sync.Mutex

	searchTimes   []time.Time
	searchLimitMu sync.Mutex
)

func isOnCooldown(userID string) bool {
	cooldownsMu.Lock()
	defer cooldownsMu.Unlock()
	if t, ok := cooldowns[userID]; ok && time.Since(t) < userCooldown {
		return true
	}
	cooldowns[userID] = time.Now()
	return false
}

func isSearchRateLimited() bool {
	searchLimitMu.Lock()
	defer searchLimitMu.Unlock()
	now := time.Now()
	cutoff := now.Add(-globalSearchWindow)
	valid := searchTimes[:0]
	for _, t := range searchTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	searchTimes = valid
	if len(searchTimes) >= globalSearchLimit {
		return true
	}
	searchTimes = append(searchTimes, now)
	return false
}

func isDMOnCooldown() bool {
	dmCooldownMu.Lock()
	defer dmCooldownMu.Unlock()
	if time.Since(lastDMTime) < globalDMCooldown {
		return true
	}
	lastDMTime = time.Now()
	return false
}
