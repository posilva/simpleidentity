// Package certs manages the certificates handling
package certs

import (
	"crypto/rsa"
	"time"
)

// CacheManager defines the interface of the cache manager for certificates
type CacheManager interface {
	Get(id string) *rsa.PublicKey
	Add(id string, pub *rsa.PublicKey, expiresAt time.Time) error
	Reset() error
}

type cacheEntry struct {
	pubKey    *rsa.PublicKey
	expiresAt int64
}

// SimpleCacheManager implements the CacheManager interface
type simpleCacheManager struct {
	cache map[string]cacheEntry
}

func NewSimpleCacheManager() CacheManager {
	return &simpleCacheManager{
		cache: make(map[string]cacheEntry, 5),
	}
}

func (cm *simpleCacheManager) Get(id string) *rsa.PublicKey {
	e, ok := cm.cache[id]
	if ok {
		if time.Now().Unix() < e.expiresAt {
			return e.pubKey
		}
	}

	return nil
}

func (cm *simpleCacheManager) Add(id string, pub *rsa.PublicKey, expiresAt time.Time) error {
	cm.cache[id] = cacheEntry{
		pubKey:    pub,
		expiresAt: expiresAt.UTC().Unix(),
	}
	return nil
}

func (cm *simpleCacheManager) Reset() error {
	for k := range cm.cache {
		delete(cm.cache, k)
	}

	return nil
}
