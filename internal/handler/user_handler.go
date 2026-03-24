package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sheranthaperera93/r2-notify-server/internal/middleware"
	userRepo "github.com/sheranthaperera93/r2-notify-server/internal/repository/user"
)

type UserHandler struct {
	userRepo userRepo.UserRepository
}

func NewUserHandler(userRepo userRepo.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"verified":   user.Verified,
		"created_at": user.CreatedAt,
	})
}
