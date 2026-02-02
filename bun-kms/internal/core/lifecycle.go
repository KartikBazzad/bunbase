package core

import "time"

// IsRevoked returns true if the key has been revoked.
func IsRevoked(key *Key) bool {
	return key != nil && key.RevokedAt != nil
}

// RotationPolicy defines when a key should be rotated (for future use).
type RotationPolicy struct {
	TTLDays int // Rotate when key is older than TTL days; 0 = no automatic rotation
}

// ShouldRotate returns true if the key's latest version is older than the policy TTL.
func (p *RotationPolicy) ShouldRotate(key *Key) bool {
	if p == nil || p.TTLDays <= 0 || key == nil || len(key.Versions) == 0 {
		return false
	}
	latest := key.Versions[len(key.Versions)-1].CreatedAt
	cutoff := time.Now().UTC().AddDate(0, 0, -p.TTLDays)
	return latest.Before(cutoff)
}
