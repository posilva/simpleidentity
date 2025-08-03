package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func genPubKey(t *testing.T) *rsa.PublicKey {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return &privateKey.PublicKey
}

func TestCache_SimpleCacheManager_Returns_ID(t *testing.T) {
	cm := NewSimpleCacheManager()
	err := cm.Add("good-pub-key", genPubKey(t), time.Now().Add(10+time.Second).UTC())
	require.Nil(t, err)
	k := cm.Get("good-pub-key")
	require.NotNil(t, k)
}

func TestCache_SimpleCacheManager_Returns_Nil_NotFound(t *testing.T) {
	cm := NewSimpleCacheManager()
	k := cm.Get("does not exist")
	require.Nil(t, k)
}

func TestCache_SimpleCacheManager_Returns_Nil_WhenEntryExpired(t *testing.T) {
	cm := NewSimpleCacheManager()
	err := cm.Add("good-pub-key", genPubKey(t), time.Now().Add(-10*time.Second).UTC())
	require.Nil(t, err)
	k := cm.Get("good-pub-key")
	require.Nil(t, k)
}
