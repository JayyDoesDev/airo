package events

import "github.com/jayydoesdev/airo/bot/trust"

const (
	TierPrimeAdmin = trust.TierPrimeAdmin
	TierTrusted    = trust.TierTrusted
	TierRegular    = trust.TierRegular
	TierBlocked    = trust.TierBlocked
)

func LoadTiers()                        { trust.Load() }
func GetTier(userID string) int         { return trust.GetTier(userID) }
func SetTier(userID string, tier int) error { return trust.SetTier(userID, tier) }
func TierLabel(tier int) string         { return trust.Label(tier) }
