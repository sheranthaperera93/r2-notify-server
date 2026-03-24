package keyService

import (
	"context"
	"errors"
	"fmt"

	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	userRepo "github.com/sheranthaperera93/r2-notify-server/internal/repository/user"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

type CreatedKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"` // shown once, never stored
	Name  string `json:"name"`
}

type KeyBasic struct {
	KeyID      string  `json:"key_id"`
	Start      string  `json:"start"`
	Enabled    bool    `json:"enabled"`
	Name       *string `json:"name"`
	Created    int64   `json:"created"`
	LastUsedAt *int64  `json:"last_used_at"`
}

type KeyDetail struct {
	KeyID            string   `json:"key_id"`
	Start            string   `json:"start"`
	Enabled          bool     `json:"enabled"`
	Name             *string  `json:"name"`
	Created          int64    `json:"created"`
	LastUsedAt       *int64   `json:"last_used"`
	UpdatedAt        *int64   `json:"updated"`
	Expires          *int64   `json:"expires"`
	Permissions      []string `json:"permissions"`
	Roles            []string `json:"roles"`
	CreditsRemaining *int64   `json:"requests_remaining"`
}

type KeyService struct {
	userRepo userRepo.UserRepository
}

func NewKeyService(userRepo userRepo.UserRepository) *KeyService {
	return &KeyService{userRepo: userRepo}
}

func (s *KeyService) CreateKey(userName, name string) (*CreatedKey, error) {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByUsername(userName)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))

	// Lazily create Unkey Identity if missing.
	if user.ExternalId == "" {
		resp, err := unkeyClient.Identities.CreateIdentity(
			context.Background(),
			components.V2IdentitiesCreateIdentityRequestBody{ExternalID: user.Username},
		)
		if err != nil || resp.V2IdentitiesCreateIdentityResponseBody == nil {
			return nil, fmt.Errorf("failed to create Unkey identity: %w", err)
		}
		externalID := resp.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID
		if err := s.userRepo.SetExternalID(user.ID, externalID); err != nil {
			logger.Log.Warn(logger.LogPayload{
				Component: "KeyService",
				Operation: "CreateKey",
				Message:   "failed to persist external ID for user " + user.ID,
				Error:     err,
			})
		}
		user.ExternalId = externalID
	}

	existing, err := s.ListKeys(userName)
	if err != nil {
		return nil, err
	}
	if len(existing) >= cfg.AllowedAPIKeyCount {
		return nil, fmt.Errorf("maximum of %d keys allowed per account", cfg.AllowedAPIKeyCount)
	}

	resp, err := unkeyClient.Keys.CreateKey(
		context.Background(),
		components.V2KeysCreateKeyRequestBody{
			APIID:      cfg.UnkeyAPIID,
			Name:       unkey.Pointer(name),
			ExternalID: unkey.Pointer(user.ExternalId),
			Prefix:     unkey.Pointer("r2"),
		},
	)
	if err != nil || resp.V2KeysCreateKeyResponseBody == nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	return &CreatedKey{
		KeyID: resp.V2KeysCreateKeyResponseBody.Data.KeyID,
		Key:   resp.V2KeysCreateKeyResponseBody.Data.Key,
		Name:  name,
	}, nil
}

func (s *KeyService) ListKeys(userName string) ([]KeyBasic, error) {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByUsername(userName)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if user.ExternalId == "" {
		return []KeyBasic{}, nil
	}

	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))

	resp, err := unkeyClient.Apis.ListKeys(
		context.Background(),
		components.V2ApisListKeysRequestBody{
			APIID:      cfg.UnkeyAPIID,
			ExternalID: unkey.Pointer(user.ExternalId),
		},
	)
	if err != nil || resp.V2ApisListKeysResponseBody == nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	keys := make([]KeyBasic, len(resp.V2ApisListKeysResponseBody.Data))
	for i, k := range resp.V2ApisListKeysResponseBody.Data {
		keys[i] = KeyBasic{
			KeyID:      k.KeyID,
			Created:    k.CreatedAt,
			Name:       k.Name,
			LastUsedAt: k.LastUsedAt,
			Start:      k.Start,
			Enabled:    k.Enabled,
		}
	}
	return keys, nil
}

func (s *KeyService) RevokeKey(userName, keyID string) error {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByUsername(userName)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.ExternalId == "" {
		return errors.New("no keys found for this user")
	}

	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))

	if err := s.verifyOwnership(unkeyClient, cfg.UnkeyAPIID, user.ExternalId, keyID); err != nil {
		return err
	}

	_, err = unkeyClient.Keys.DeleteKey(
		context.Background(),
		components.V2KeysDeleteKeyRequestBody{KeyID: keyID},
	)
	return err
}

func (s *KeyService) GetKeyDetails(userID, keyID string) (*KeyDetail, error) {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if user.ExternalId == "" {
		return nil, errors.New("no keys found for this user")
	}

	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))

	if err := s.verifyOwnership(unkeyClient, cfg.UnkeyAPIID, user.ExternalId, keyID); err != nil {
		return nil, err
	}

	resp, err := unkeyClient.Keys.GetKey(
		context.Background(),
		components.V2KeysGetKeyRequestBody{KeyID: keyID},
	)
	if err != nil || resp.V2KeysGetKeyResponseBody == nil {
		return nil, fmt.Errorf("failed to get key details: %w", err)
	}

	k := resp.V2KeysGetKeyResponseBody.Data
	detail := &KeyDetail{
		KeyID:       k.KeyID,
		Start:       k.Start,
		Enabled:     k.Enabled,
		Name:        k.Name,
		Created:     k.CreatedAt,
		LastUsedAt:  k.LastUsedAt,
		UpdatedAt:   k.UpdatedAt,
		Expires:     k.Expires,
		Permissions: k.Permissions,
		Roles:       k.Roles,
	}
	if k.Credits != nil && k.Credits.Remaining != nil {
		detail.CreditsRemaining = k.Credits.Remaining
	}
	return detail, nil
}

func (s *KeyService) UpdateKey(userID, keyID string, name *string, enabled *bool) error {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.ExternalId == "" {
		return errors.New("no keys found for this user")
	}

	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))

	if err := s.verifyOwnership(unkeyClient, cfg.UnkeyAPIID, user.ExternalId, keyID); err != nil {
		return err
	}

	updateReq := components.V2KeysUpdateKeyRequestBody{KeyID: keyID}
	if name != nil {
		updateReq.Name = name
	}
	if enabled != nil {
		updateReq.Enabled = enabled
	}

	_, err = unkeyClient.Keys.UpdateKey(context.Background(), updateReq)
	return err
}

func (s *KeyService) verifyOwnership(client *unkey.Unkey, apiID, externalID, keyID string) error {
	listResp, err := client.Apis.ListKeys(
		context.Background(),
		components.V2ApisListKeysRequestBody{
			APIID:      apiID,
			ExternalID: unkey.Pointer(externalID),
		},
	)
	if err != nil || listResp.V2ApisListKeysResponseBody == nil {
		return fmt.Errorf("failed to verify key ownership: %w", err)
	}
	for _, k := range listResp.V2ApisListKeysResponseBody.Data {
		if k.KeyID == keyID {
			return nil
		}
	}
	return errors.New("key not found or does not belong to this user")
}

func (s *KeyService) ValidateAPIKey(apiKey string) (string, error) {
	ctx := context.Background()

	unkeyClient := unkey.New(
		unkey.WithSecurity(config.LoadConfig().UnkeyRootKey),
	)

	res, err := unkeyClient.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{
		Key: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("API Key Request failed: %w", err)
	}

	body := res.V2KeysVerifyKeyResponseBody
	if body == nil {
		return "", fmt.Errorf("Empty response from API key verification")
	}

	if !body.Data.Valid {
		switch body.Data.Code {
		case "EXPIRED":
			return "", fmt.Errorf("API key has expired")
		case "DISABLED":
			return "", fmt.Errorf("API key is disabled")
		case "RATE_LIMITED":
			return "", fmt.Errorf("Rate limit exceeded")
		case "USAGE_EXCEEDED":
			return "", fmt.Errorf("Usage quota exceeded")
		default:
			return "", fmt.Errorf("invalid API key: %s", body.Data.Code)
		}
	}

	if body.Data.Identity == nil || body.Data.Identity.ExternalID == "" {
		return "", fmt.Errorf("API key has no identity set")
	}

	return body.Data.Identity.ExternalID, nil
}
