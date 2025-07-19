package ports

import (
	"context"

	"github.com/posilva/account-service/internal/core/domain"
)

type AccountsRepository interface {
	ResolveIDByProvider(context.Context, domain.ProviderType, string) (domain.AccountID, error)
	Create(context.Context, domain.ProviderType, string) (domain.AccountID, error)
}
