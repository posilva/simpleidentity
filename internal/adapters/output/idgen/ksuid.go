package idgen

import (
	"github.com/posilva/account-service/internal/core/ports"
	"github.com/segmentio/ksuid"
)

type ksuidGenerator struct{}

// NewKSUIDGenerator creates a new instance of ksuidGenerator.
func NewKSUIDGenerator() *ksuidGenerator {
	return &ksuidGenerator{}
}

var _ ports.IDGenerator = (*ksuidGenerator)(nil)

// Generate generates a new KSUID.
func (g *ksuidGenerator) GenerateID() string {
	return ksuid.New().String()
}
