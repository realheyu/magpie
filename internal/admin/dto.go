package admin

import "time"

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      userResp  `json:"user"`
}

type userResp struct {
	ID          uint64     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type appResp struct {
	ID          uint64    `json:"id"`
	AppName     string    `json:"appName"`
	Description string    `json:"description"`
	Format      string    `json:"format"`
	Content     string    `json:"content"`
	Sensitive   bool      `json:"sensitive"`
	Version     int64     `json:"version"`
	Status      string    `json:"status"`
	Permission  string    `json:"permission"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type saveAppRequest struct {
	AppName       string `json:"appName"`
	Description   string `json:"description"`
	Format        string `json:"format"`
	Content       string `json:"content"`
	Sensitive     bool   `json:"sensitive"`
	Status        string `json:"status"`
	ChangeSummary string `json:"changeSummary"`
}

type revisionResp struct {
	ID              uint64    `json:"id"`
	AppName         string    `json:"appName"`
	Version         int64     `json:"version"`
	Format          string    `json:"format"`
	Content         string    `json:"content"`
	Sensitive       bool      `json:"sensitive"`
	ChangeSummary   string    `json:"changeSummary"`
	CreatedByUserID *uint64   `json:"createdByUserId"`
	CreatedAt       time.Time `json:"createdAt"`
}

type rollbackRequest struct {
	Version int64 `json:"version" binding:"required"`
}

type createUserRequest struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password" binding:"required"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type updateUserRequest struct {
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type permissionEntry struct {
	AppName    string `json:"appName" binding:"required"`
	Permission string `json:"permission" binding:"required"`
}

type permissionsRequest struct {
	Permissions []permissionEntry `json:"permissions"`
}

type permissionsResponse struct {
	Permissions []permissionEntry `json:"permissions"`
}

type apiKeyResp struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	KeyPreview string     `json:"keyPreview"`
	Status     string     `json:"status"`
	AppNames   []string   `json:"appNames"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type createAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	AppNames  []string   `json:"appNames" binding:"required"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type createAPIKeyResponse struct {
	APIKey string     `json:"apiKey"`
	Item   apiKeyResp `json:"item"`
}

type updateAPIKeyAppsRequest struct {
	AppNames []string `json:"appNames" binding:"required"`
}
