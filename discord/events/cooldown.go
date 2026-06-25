package events

import (
	"sync"
	"time"
)

const userCooldown = 5 * time.Second
const maxDMsPerRequest = 1

var (
	cooldowns   = map[string]time.Time{}
	cooldownsMu sync.Mutex
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
