package tokenRepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TokenRepository interface {
	CreateRefreshToken(userID string, expiry time.Duration) (string, error)
	ConsumeRefreshToken(token string) (string, error)
	DeleteRefreshToken(token string) error
	DeleteAllRefreshTokens(userID string) error

	CreateVerifyToken(userID string) (string, error)
	ConsumeVerifyToken(token string) (string, error)

	CreateResetToken(userID string) (string, error)
	ConsumeResetToken(token string) (string, error)
}

type tokenRepositoryImpl struct {
	db *mongo.Database
}

func NewTokenRepository(db *mongo.Database) TokenRepository {
	return &tokenRepositoryImpl{db: db}
}

func (r *tokenRepositoryImpl) col(name string) *mongo.Collection {
	return r.db.Collection(name)
}

// --- Refresh Tokens ---

func (r *tokenRepositoryImpl) CreateRefreshToken(userID string, expiry time.Duration) (string, error) {
	tok := uuid.New().String()
	_, err := r.col("refresh_tokens").InsertOne(context.Background(), bson.M{
		"token":      tok,
		"user_id":    userID,
		"expires_at": time.Now().Add(expiry),
	})
	if err != nil {
		return "", err
	}
	return tok, nil
}

func (r *tokenRepositoryImpl) ConsumeRefreshToken(token string) (string, error) {
	return r.consumeToken("refresh_tokens", token)
}

func (r *tokenRepositoryImpl) DeleteRefreshToken(token string) error {
	_, err := r.col("refresh_tokens").DeleteOne(context.Background(), bson.M{"token": token})
	return err
}

func (r *tokenRepositoryImpl) DeleteAllRefreshTokens(userID string) error {
	_, err := r.col("refresh_tokens").DeleteMany(context.Background(), bson.M{"user_id": userID})
	return err
}

// --- Email Verification Tokens ---

func (r *tokenRepositoryImpl) CreateVerifyToken(userID string) (string, error) {
	// Only one active verification token per user
	_, _ = r.col("verification_tokens").DeleteMany(context.Background(), bson.M{"user_id": userID})
	tok := uuid.New().String()
	_, err := r.col("verification_tokens").InsertOne(context.Background(), bson.M{
		"token":      tok,
		"user_id":    userID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		return "", err
	}
	return tok, nil
}

func (r *tokenRepositoryImpl) ConsumeVerifyToken(token string) (string, error) {
	return r.consumeToken("verification_tokens", token)
}

// --- Password Reset Tokens ---

func (r *tokenRepositoryImpl) CreateResetToken(userID string) (string, error) {
	// Only one active reset token per user
	_, _ = r.col("password_reset_tokens").DeleteMany(context.Background(), bson.M{"user_id": userID})
	tok := uuid.New().String()
	_, err := r.col("password_reset_tokens").InsertOne(context.Background(), bson.M{
		"token":      tok,
		"user_id":    userID,
		"expires_at": time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		return "", err
	}
	return tok, nil
}

func (r *tokenRepositoryImpl) ConsumeResetToken(token string) (string, error) {
	return r.consumeToken("password_reset_tokens", token)
}

// consumeToken finds, validates expiry, then deletes a token document.
// Returns the owning userID if valid.
func (r *tokenRepositoryImpl) consumeToken(collection, token string) (string, error) {
	var doc bson.M
	err := r.col(collection).FindOneAndDelete(
		context.Background(),
		bson.M{"token": token},
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", errors.New("token not found or already used")
	}
	if err != nil {
		return "", err
	}

	expiresAt, ok := doc["expires_at"].(time.Time)
	if !ok {
		return "", errors.New("invalid token document")
	}
	if time.Now().After(expiresAt) {
		return "", errors.New("token has expired")
	}

	userID, _ := doc["user_id"].(string)
	return userID, nil
}
