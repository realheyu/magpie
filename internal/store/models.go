package store

import (
	"time"

	"gorm.io/gorm"
)

type App struct {
	ID          uint64         `gorm:"primaryKey"`
	AppName     string         `gorm:"size:128;not null;uniqueIndex"`
	Description string         `gorm:"size:512"`
	Format      string         `gorm:"size:32;not null;default:toml"`
	Content     string         `gorm:"type:longtext"`
	Sensitive   bool           `gorm:"not null;default:false"`
	Version     int64          `gorm:"not null;default:1"`
	Status      string         `gorm:"size:32;not null;default:active;index"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type AppRevision struct {
	ID              uint64    `gorm:"primaryKey"`
	AppID           uint64    `gorm:"not null;index"`
	AppName         string    `gorm:"size:128;not null;index"`
	Version         int64     `gorm:"not null;index"`
	Format          string    `gorm:"size:32;not null"`
	Content         string    `gorm:"type:longtext"`
	Sensitive       bool      `gorm:"not null;default:false"`
	ChangeSummary   string    `gorm:"size:512"`
	CreatedByUserID *uint64   `gorm:"index"`
	CreatedAt       time.Time `gorm:"not null"`
}

type User struct {
	ID           uint64     `gorm:"primaryKey"`
	Username     string     `gorm:"size:64;not null;uniqueIndex"`
	DisplayName  string     `gorm:"size:128"`
	PasswordHash string     `gorm:"size:128;not null"`
	PasswordSalt string     `gorm:"size:64;not null"`
	Role         string     `gorm:"size:32;not null;default:user;index"`
	Status       string     `gorm:"size:32;not null;default:active;index"`
	LastLoginAt  *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

type UserAppPermission struct {
	ID         uint64    `gorm:"primaryKey"`
	UserID     uint64    `gorm:"not null;uniqueIndex:idx_user_app"`
	AppID      uint64    `gorm:"not null;uniqueIndex:idx_user_app"`
	Permission string    `gorm:"size:32;not null"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

type APIKey struct {
	ID         uint64     `gorm:"primaryKey"`
	Name       string     `gorm:"size:128;not null"`
	KeyHash    string     `gorm:"size:128;not null;uniqueIndex"`
	KeyPreview string     `gorm:"size:32;not null"`
	Status     string     `gorm:"size:32;not null;default:active;index"`
	ExpiresAt  *time.Time `gorm:"index"`
	LastUsedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
	UpdatedAt  time.Time  `gorm:"not null"`
}

type APIKeyApp struct {
	ID        uint64    `gorm:"primaryKey"`
	APIKeyID  uint64    `gorm:"not null;uniqueIndex:idx_api_key_app"`
	AppID     uint64    `gorm:"not null;uniqueIndex:idx_api_key_app"`
	CreatedAt time.Time `gorm:"not null"`
}

type AuditLog struct {
	ID           uint64    `gorm:"primaryKey"`
	ActorType    string    `gorm:"size:32;not null;index"`
	ActorID      *uint64   `gorm:"index"`
	Action       string    `gorm:"size:64;not null;index"`
	ResourceType string    `gorm:"size:64;not null;index"`
	ResourceID   string    `gorm:"size:128;not null;index"`
	Metadata     string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"not null;index"`
}
