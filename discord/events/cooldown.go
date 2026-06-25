package events

import (
	"sync"
	"time"
)

const userCooldown = 5 * time.Second
const maxDMsPerRequest = 1
const globalDMCooldown = 10 * time.Second

var (
	cooldowns    = map[string]time.Time{}
	cooldownsMu  sync.Mutex
	lastDMTime   time.Time
	dmCooldownMu sync.Mutex
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

func isDMOnCooldown() bool {
	dmCooldownMu.Lock()
	defer dmCooldownMu.Unlock()
	if time.Since(lastDMTime) < globalDMCooldown {
		return true
	}
	lastDMTime = time.Now()
	return false
}
