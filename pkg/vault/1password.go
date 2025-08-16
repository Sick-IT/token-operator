package vault

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hamba/cmd/v3/observe"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/sethvargo/go-retry"
	"gitlab.com/sickit/test-public/pkg/token"
)

const (
	Type1Password      = "1password"
	IntegrationName    = "token-operator"
	IntegrationVersion = "v0.1.0"
)

func NewOnePasswordVault(ctx context.Context, token string, obsvr *observe.Observer) (*OnePassword, error) {
	op, err := onepassword.NewClient(ctx, onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo(IntegrationName, IntegrationVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1password client: %w", err)
	}

	b := retry.NewExponential(50 * time.Millisecond)
	b = retry.WithMaxRetries(10, b)
	b = retry.WithMaxDuration(30*time.Second, b)

	return &OnePassword{
		client:  op,
		backoff: b,
		log:     obsvr.Log,
		ctx:     ctx,
	}, nil
}

type OnePassword struct {
	client  *onepassword.Client
	dryRun  bool
	backoff retry.Backoff

	log *logger.Logger
	ctx context.Context
}

func (o *OnePassword) WithDryRun(dryRun bool) {
	o.dryRun = dryRun
}

func (o *OnePassword) GetItem(item *token.Vault) (*Item, error) {
	path := fmt.Sprintf("op://%s/%s/%s", item.Path, item.Item, item.Field)
	o.log.Info("checking 1password vault item", lctx.Str("path", path))

	b := o.backoff
	secret := ""
	err := retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		secret, err = o.client.Secrets().Resolve(o.ctx, path)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		if !strings.Contains(err.Error(), "no item matched") {
			return nil, fmt.Errorf("failed to resolve 1password vault item: %w", err)
		}
		o.log.Debug("secret not found", lctx.Err(err))
		return nil, ErrItemNotFound
	}

	return &Item{
		Name:  item.Item,
		Path:  item.Path,
		Field: item.Field,
		Value: secret,
	}, nil
}

func (o *OnePassword) CreateItem(vault *token.Vault, value string) (*Item, error) {
	opvault, err := o.findVault(vault)
	if err != nil {
		return nil, fmt.Errorf("failed to find 1password vault: %w", err)
	}

	o.log.Debug("creating item in 1password vault", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
	if o.dryRun {
		o.log.Info("dry-run flag set, not creating 1password vault item", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
		return &Item{
			Name:  vault.Item,
			Path:  opvault.ID,
			Field: vault.Field,
			Value: value,
		}, nil
	}

	create := onepassword.ItemCreateParams{
		Category: onepassword.ItemCategoryLogin,
		// only works with the 1password identifier
		VaultID: opvault.ID,
		Title:   vault.Item,
		Fields: []onepassword.ItemField{
			{
				ID:        vault.Field,
				Title:     vault.Field,
				Value:     value,
				FieldType: onepassword.ItemFieldTypeConcealed,
			},
		},
	}

	b := o.backoff
	opitem := onepassword.Item{}
	err = retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		opitem, err = o.client.Items().Create(o.ctx, create)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create 1password vault item: %w", err)
	}

	o.log.Debug("created item in 1password vault",
		lctx.Str("vault", opitem.VaultID),
		lctx.Str("item", vault.Item),
		lctx.Uint32("version", opitem.Version),
	)

	return &Item{
		Name:  opitem.Title,
		Path:  opitem.VaultID,
		Field: opitem.Fields[0].Title,
		Value: opitem.Fields[0].Value,
	}, nil
}

func (o *OnePassword) UpdateItem(vault *token.Vault, value string) error {
	opvault, err := o.findVault(vault)
	if err != nil {
		return fmt.Errorf("failed to find 1password vault: %w", err)
	}

	opitem, err := o.findItem(opvault.ID, vault)
	if err != nil {
		return fmt.Errorf("failed to find 1password vault item: %w", err)
	}

	b := o.backoff
	update := onepassword.Item{}
	err = retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		update, err = o.client.Items().Get(o.ctx, opvault.ID, opitem.ID)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get 1password vault item: %w", err)
	}

	// update value of matching field
	for i, field := range update.Fields {
		if field.Title == vault.Field {
			update.Fields[i].Value = value
			break
		}
	}

	o.log.Debug("updating item in 1password vault", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
	if o.dryRun {
		o.log.Info("dry-run flag set, not updating 1password vault item", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
		return nil
	}

	b = o.backoff
	err = retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		_, err = o.client.Items().Put(o.ctx, update)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update 1password vault item: %w", err)
	}

	return nil
}

func (o *OnePassword) DeleteItem(vault *token.Vault) error {
	opvault, err := o.findVault(vault)
	if err != nil {
		return fmt.Errorf("failed to find 1password vault: %w", err)
	}

	opitem, err := o.findItem(opvault.ID, vault)
	if err != nil {
		return fmt.Errorf("failed to find 1password vault item: %w", err)
	}

	o.log.Debug("deleting item in 1password vault", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
	if o.dryRun {
		o.log.Info("dry-run flag set, not deleting 1password vault item", lctx.Str("vault", opvault.ID), lctx.Str("item", vault.Item))
		return nil
	}

	b := o.backoff
	err = retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err = o.client.Items().Delete(o.ctx, opvault.ID, opitem.ID)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete 1password vault item: %w", err)
	}

	return nil
}

func (o *OnePassword) findVault(item *token.Vault) (*onepassword.VaultOverview, error) {
	b := o.backoff
	vaults := []onepassword.VaultOverview{}
	err := retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		vaults, err = o.client.Vaults().List(o.ctx)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list 1password vaults: %w", err)
	}

	vault := onepassword.VaultOverview{}
	for _, vlt := range vaults {
		if vlt.Title == item.Path {
			vault = vlt
			break
		}
	}

	if vault.ID == "" {
		return nil, ErrVaultNotFound
	}

	return &vault, nil
}

func (o *OnePassword) findItem(vaultID string, item *token.Vault) (*onepassword.ItemOverview, error) {
	b := o.backoff
	opitems := []onepassword.ItemOverview{}
	err := retry.Do(o.ctx, b, func(ctx context.Context) error {
		var err error
		opitems, err = o.client.Items().List(o.ctx, vaultID)
		if retryErr := o.isRetriable(err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list 1password items: %w", err)
	}

	opitem := onepassword.ItemOverview{}
	for _, itm := range opitems {
		if itm.Title == item.Item {
			opitem = itm
		}
	}

	if opitem.ID == "" {
		return nil, ErrItemNotFound
	}

	return &opitem, nil
}

func (o *OnePassword) isRetriable(err error) error {
	if err == nil {
		return nil
	}

	// TODO: can't we match errors better than through string?
	switch {
	case strings.HasPrefix(err.Error(), "error resolving secret reference"):
		return err
	}

	o.log.Debug("retry on err", lctx.Err(err))
	return retry.RetryableError(err)
}
