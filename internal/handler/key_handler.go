package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sheranthaperera93/r2-notify-server/internal/middleware"
	"github.com/sheranthaperera93/r2-notify-server/internal/models"
	keyService "github.com/sheranthaperera93/r2-notify-server/internal/services/key"
)

type KeyHandler struct {
	keySvc *keyService.KeyService
}

func NewKeyHandler(keySvc *keyService.KeyService) *KeyHandler {
	return &KeyHandler{keySvc: keySvc}
}

func (h *KeyHandler) CreateKey(c *gin.Context) {
	var req models.CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userName := middleware.GetUserName(c)
	key, err := h.keySvc.CreateKey(userName, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, key)
}

func (h *KeyHandler) ListKeys(c *gin.Context) {
	userName := middleware.GetUserName(c)
	keys, err := h.keySvc.ListKeys(userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func (h *KeyHandler) RevokeKey(c *gin.Context) {
	keyID := c.Param("keyId")
	userName := middleware.GetUserName(c)
	if err := h.keySvc.RevokeKey(userName, keyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Key revoked successfully."})
}

func (h *KeyHandler) GetKeyDetails(c *gin.Context) {
	keyID := c.Param("keyId")
	userID := middleware.GetUserID(c)
	detail, err := h.keySvc.GetKeyDetails(userID, keyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *KeyHandler) UpdateKey(c *gin.Context) {
	keyID := c.Param("keyId")
	userID := middleware.GetUserID(c)
	var req models.UpdateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.keySvc.UpdateKey(userID, keyID, req.Name, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Key updated successfully."})
}
