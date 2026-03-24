package userRepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicateEmail = errors.New("email already registered")
var ErrDuplicateUsername = errors.New("username already taken")

type UserRepository interface {
	Create(username, email, passwordHash string) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	FindByID(id string) (*models.User, error)
	SetVerified(userID string) error
	SetExternalID(userID, externalID string) error
	UpdatePassword(userID, passwordHash string) error
}

type userRepositoryImpl struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) col() *mongo.Collection {
	return r.db.Collection("users")
}

func (r *userRepositoryImpl) Create(username, email, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Verified:     false,
		CreatedAt:    time.Now(),
	}

	_, err := r.col().InsertOne(context.Background(), bson.M{
		"_id":           user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"verified":      user.Verified,
		"external_id":   "",
		"created_at":    user.CreatedAt,
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if containsStr(err.Error(), "username") {
				return nil, ErrDuplicateUsername
			}
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	logger.Log.Info(logger.LogPayload{
		Component: "UserRepository",
		Operation: "Create",
		Message:   "User created: " + user.ID,
	})
	return user, nil
}

func (r *userRepositoryImpl) FindByEmail(email string) (*models.User, error) {
	return r.findOne(bson.M{"email": email})
}

func (r *userRepositoryImpl) FindByUsername(username string) (*models.User, error) {
	return r.findOne(bson.M{"username": username})
}

func (r *userRepositoryImpl) FindByID(id string) (*models.User, error) {
	return r.findOne(bson.M{"_id": id})
}

func (r *userRepositoryImpl) findOne(filter bson.M) (*models.User, error) {
	var raw bson.M
	err := r.col().FindOne(context.Background(), filter).Decode(&raw)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapUser(raw), nil
}

func (r *userRepositoryImpl) SetVerified(userID string) error {
	_, err := r.col().UpdateOne(context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"verified": true}},
	)
	return err
}

func (r *userRepositoryImpl) SetExternalID(userID, externalID string) error {
	_, err := r.col().UpdateOne(context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"external_id": externalID}},
	)
	return err
}

func (r *userRepositoryImpl) UpdatePassword(userID, passwordHash string) error {
	_, err := r.col().UpdateOne(context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"password_hash": passwordHash}},
	)
	return err
}

func mapUser(raw bson.M) *models.User {
	u := &models.User{}
	if v, ok := raw["_id"].(string); ok {
		u.ID = v
	}
	if v, ok := raw["username"].(string); ok {
		u.Username = v
	}
	if v, ok := raw["email"].(string); ok {
		u.Email = v
	}
	if v, ok := raw["password_hash"].(string); ok {
		u.PasswordHash = v
	}
	if v, ok := raw["verified"].(bool); ok {
		u.Verified = v
	}
	if v, ok := raw["external_id"].(string); ok {
		u.ExternalId = v
	}
	if v, ok := raw["created_at"].(time.Time); ok {
		u.CreatedAt = v
	}
	return u
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s[1:], sub) || s[:len(sub)] == sub)
}
