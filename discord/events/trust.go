package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const (
	TierPrimeAdmin = 0
	TierTrusted    = 1
	TierRegular    = 2
	TierBlocked    = 3
)

var (
	userTiers   = map[string]int{}
	userTiersMu sync.RWMutex
)

const tierFile = "user_tiers.json"

func LoadTiers() {
	data, err := os.ReadFile(tierFile)
	if err != nil {
		return
	}
	userTiersMu.Lock()
	json.Unmarshal(data, &userTiers)
	userTiersMu.Unlock()
}

func saveTiers() {
	userTiersMu.RLock()
	data, err := json.Marshal(userTiers)
	userTiersMu.RUnlock()
	if err != nil {
		return
	}
	os.WriteFile(tierFile, data, 0644)
}

func GetTier(userID string) int {
	if userID == "419958345487745035" {
		return TierPrimeAdmin
	}
	userTiersMu.RLock()
	t, ok := userTiers[userID]
	userTiersMu.RUnlock()
	if !ok {
		return TierRegular
	}
	return t
}

func SetTier(userID string, tier int) error {
	if userID == "419958345487745035" {
		return fmt.Errorf("cannot change prime admin tier")
	}
	if tier < TierTrusted || tier > TierBlocked {
		return fmt.Errorf("invalid tier %d", tier)
	}
	userTiersMu.Lock()
	userTiers[userID] = tier
	userTiersMu.Unlock()
	saveTiers()
	return nil
}

func TierLabel(tier int) string {
	switch tier {
	case TierPrimeAdmin:
		return "Prime Admin"
	case TierTrusted:
		return "Trusted"
	case TierBlocked:
		return "Blocked"
	default:
		return "Regular"
	}
}
