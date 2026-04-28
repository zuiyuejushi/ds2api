package account

import "context"

// PoolInterface defines the shared contract for account pools (local and remote).
type PoolInterface interface {
	AcquireWait(ctx context.Context, target string, exclude map[string]bool) (*PoolAccount, error)
	Release(accountID string) error
	Reset()
	Stats() PoolStats
}

// PoolAccount represents an acquired account with its token.
type PoolAccount struct {
	ID    string
	Token string
}

// PoolStats holds pool usage statistics.
type PoolStats struct {
	Total     int
	InUse     int
	Available int
	QueueLen  int
}
