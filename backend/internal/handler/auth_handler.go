package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg         *config.Config
	settingSvc  *service.SettingService
	userService *service.UserService
}

func NewAuthHandler(cfg *config.Config, settings *service.SettingService, users *service.UserService) *AuthHandler {
	return &AuthHandler{cfg: cfg, settingSvc: settings, userService: users}
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	profile, err := h.userService.GetByID(c.Request.Context(), user.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	profile.Role = user.Role
	profile.Email = user.Email
	response.Success(c, struct {
		userProfileResponse
		RunMode string `json:"run_mode"`
	}{userProfileResponse{User: *dto.UserFromService(profile), AvatarURL: profile.AvatarURL}, h.cfg.RunMode})
}
