package filetoken

import "sync"

var cache sync.Map // fileID → int (token count)

// Store saves the token count for a file ID.
func Store(fileID string, tokenCount int) {
	cache.Store(fileID, tokenCount)
}

// Get retrieves the token count for a file ID. Returns 0 if not found.
func Get(fileID string) int {
	v, ok := cache.Load(fileID)
	if !ok {
		return 0
	}
	return v.(int)
}

// Remove deletes a file ID from the cache.
func Remove(fileID string) {
	cache.Delete(fileID)
}
