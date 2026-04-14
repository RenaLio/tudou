package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateUserRequest struct {
	Username string              `json:"username" binding:"required"`
	Password string              `json:"password" binding:"required"`
	Email    string              `json:"email"`
	Phone    string              `json:"phone"`
	Nickname string              `json:"nickname"`
	Avatar   string              `json:"avatar"`
	Status   *models.UserStatus  `json:"status,omitempty"`
	Role     *models.UserRole    `json:"role,omitempty"`
	Settings models.UserSettings `json:"settings"`
}

type UpdateUserRequest struct {
	Email       *string              `json:"email,omitempty"`
	Phone       *string              `json:"phone,omitempty"`
	Nickname    *string              `json:"nickname,omitempty"`
	Avatar      *string              `json:"avatar,omitempty"`
	Status      *models.UserStatus   `json:"status,omitempty"`
	Role        *models.UserRole     `json:"role,omitempty"`
	Settings    *models.UserSettings `json:"settings,omitempty"`
	LastLoginIP *string              `json:"lastLoginIP,omitempty"`
}

type UpdateUserPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

type SetUserStatusRequest struct {
	Status models.UserStatus `json:"status" binding:"required"`
}

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResponse struct {
	AccessToken string       `json:"accessToken"`
	ExpiresIn   int64        `json:"expiresIn"`
	User        UserResponse `json:"user"`
}

type ListUsersRequest struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"pageSize"`
	OrderBy       string `form:"orderBy"`
	Keyword       string `form:"keyword"`
	Status        string `form:"status"`
	Role          string `form:"role"`
	PreloadTokens bool   `form:"preloadTokens"`
	PreloadStats  bool   `form:"preloadStats"`
}

type UserResponse struct {
	ID          int64               `json:"id,string"`
	Username    string              `json:"username"`
	Email       string              `json:"email"`
	Phone       string              `json:"phone"`
	Nickname    string              `json:"nickname"`
	Avatar      string              `json:"avatar"`
	Status      models.UserStatus   `json:"status"`
	Role        models.UserRole     `json:"role"`
	LastLoginAt *time.Time          `json:"lastLoginAt,omitempty"`
	LastLoginIP string              `json:"lastLoginIP,omitempty"`
	LoginCount  int64               `json:"loginCount"`
	Settings    models.UserSettings `json:"settings"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type UserWithRelationsResponse struct {
	UserResponse
	Tokens []TokenResponse   `json:"tokens,omitempty"`
	Stats  *models.UserStats `json:"stats,omitempty"`
}
