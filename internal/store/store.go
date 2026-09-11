package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/realheyu/magpie/internal/domain"
	"github.com/realheyu/magpie/internal/security"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrNotFound = gorm.ErrRecordNotFound

type Store struct {
	db *gorm.DB
}

func Open(dsn string) (*Store, error) {
	return OpenWithLogger(dsn, nil)
}

func OpenWithLogger(dsn string, gormLogger logger.Interface) (*Store, error) {
	if gormLogger == nil {
		gormLogger = logger.Default.LogMode(logger.Warn)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Exec("SET time_zone = '+00:00'").Error; err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(
		&App{},
		&AppRevision{},
		&User{},
		&UserAppPermission{},
		&APIKey{},
		&APIKeyApp{},
		&AuditLog{},
	)
}

func (s *Store) BootstrapAdmin(username, password string) error {
	var count int64
	if err := s.db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 || username == "" || password == "" {
		return nil
	}
	hash, salt, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.db.Create(&User{
		Username:     username,
		DisplayName:  username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Role:         domain.RoleAdmin,
		Status:       domain.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error
}

func (s *Store) FindUserByUsername(username string) (*User, error) {
	var user User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) FindUserByID(id uint64) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) TouchUserLogin(id uint64) error {
	now := time.Now().UTC()
	return s.db.Model(&User{}).Where("id = ?", id).Updates(map[string]any{
		"last_login_at": now,
		"updated_at":    now,
	}).Error
}

func (s *Store) ListUsers(query string, page, pageSize int) ([]User, int64, error) {
	db := s.db.Model(&User{})
	if query != "" {
		db = db.Where("username LIKE ? OR display_name LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := db.Order("id ASC").Offset(offset(page, pageSize)).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (s *Store) CreateUser(username, displayName, password, role, status string) (*User, error) {
	if role == "" {
		role = domain.RoleUser
	}
	if status == "" {
		status = domain.StatusActive
	}
	hash, salt, err := security.HashPassword(password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &User{Username: username, DisplayName: displayName, PasswordHash: hash, PasswordSalt: salt, Role: role, Status: status, CreatedAt: now, UpdatedAt: now}
	return user, s.db.Create(user).Error
}

func (s *Store) UpdateUser(id uint64, displayName, password, role, status string) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if displayName != "" {
		updates["display_name"] = displayName
	}
	if role != "" {
		updates["role"] = role
	}
	if status != "" {
		updates["status"] = status
	}
	if password != "" {
		hash, salt, err := security.HashPassword(password)
		if err != nil {
			return nil, err
		}
		updates["password_hash"] = hash
		updates["password_salt"] = salt
	}
	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.FindUserByID(id)
}

func (s *Store) DeleteUser(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&UserAppPermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&User{}, id).Error
	})
}

func (s *Store) CreateApp(app *App, actorUserID uint64, changeSummary string) error {
	now := time.Now().UTC()
	format, err := domain.ResolveFormat(app.Format)
	if err != nil {
		return err
	}
	app.Format = format
	if err := domain.ValidateContent(app.Format, app.Content); err != nil {
		return err
	}
	if app.Status == "" {
		app.Status = domain.StatusActive
	}
	app.Version = 1
	app.CreatedAt = now
	app.UpdatedAt = now
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(app).Error; err != nil {
			return err
		}
		return createRevisionAndAudit(tx, app, actorUserID, changeSummary, "app.create")
	})
}

func (s *Store) UpdateApp(appName string, description, format, content string, sensitive bool, status string, actorUserID uint64, changeSummary string) (*App, error) {
	var app App
	if err := s.db.Where("app_name = ?", appName).First(&app).Error; err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		app.Description = description
		resolvedFormat, err := domain.ResolveFormat(format)
		if err != nil {
			return err
		}
		app.Format = resolvedFormat
		if err := domain.ValidateContent(app.Format, content); err != nil {
			return err
		}
		app.Content = content
		app.Sensitive = sensitive
		if status != "" {
			app.Status = status
		}
		app.Version++
		app.UpdatedAt = time.Now().UTC()
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		return createRevisionAndAudit(tx, &app, actorUserID, changeSummary, "app.update")
	}); err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) DeleteApp(appName string, actorUserID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var app App
		if err := tx.Where("app_name = ?", appName).First(&app).Error; err != nil {
			return err
		}
		if err := tx.Where("app_id = ?", app.ID).Delete(&UserAppPermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("app_id = ?", app.ID).Delete(&APIKeyApp{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&app).Error; err != nil {
			return err
		}
		return createAudit(tx, "user", &actorUserID, "app.delete", "app", app.AppName, "")
	})
}

func (s *Store) GetAppByName(appName string) (*App, error) {
	var app App
	if err := s.db.Where("app_name = ?", appName).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) ListAppsForUser(user *User, query string, page, pageSize int) ([]App, int64, error) {
	db := s.db.Model(&App{}).Where("status <> ?", domain.StatusDisabled)
	if query != "" {
		db = db.Where("app_name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if user.Role != domain.RoleAdmin {
		db = db.Joins("JOIN user_app_permissions uap ON uap.app_id = apps.id AND uap.user_id = ?", user.ID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var apps []App
	err := db.Order("apps.app_name ASC").Offset(offset(page, pageSize)).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

func (s *Store) GetUserAppPermission(userID, appID uint64) (string, error) {
	var permission UserAppPermission
	if err := s.db.Where("user_id = ? AND app_id = ?", userID, appID).First(&permission).Error; err != nil {
		return "", err
	}
	return permission.Permission, nil
}

func (s *Store) ListUserPermissions(userID uint64) ([]UserAppPermission, []App, error) {
	var permissions []UserAppPermission
	if err := s.db.Where("user_id = ?", userID).Order("app_id ASC").Find(&permissions).Error; err != nil {
		return nil, nil, err
	}
	appIDs := make([]uint64, 0, len(permissions))
	for _, permission := range permissions {
		appIDs = append(appIDs, permission.AppID)
	}
	var apps []App
	if len(appIDs) > 0 {
		if err := s.db.Where("id IN ?", appIDs).Find(&apps).Error; err != nil {
			return nil, nil, err
		}
	}
	return permissions, apps, nil
}

func (s *Store) ReplaceUserPermissions(userID uint64, entries map[string]string, actorUserID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&UserAppPermission{}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for appName, permission := range entries {
			if !domain.ValidPermission(permission) {
				return fmt.Errorf("应用 %s 的权限无效", appName)
			}
			var app App
			if err := tx.Where("app_name = ?", appName).First(&app).Error; err != nil {
				return err
			}
			if err := tx.Create(&UserAppPermission{UserID: userID, AppID: app.ID, Permission: permission, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return err
			}
		}
		return createAudit(tx, "user", &actorUserID, "user.permissions.update", "user", fmt.Sprint(userID), "")
	})
}

func (s *Store) ListAppRevisions(appName string, page, pageSize int) ([]AppRevision, int64, error) {
	db := s.db.Model(&AppRevision{}).Where("app_name = ?", appName)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var revisions []AppRevision
	err := db.Order("version DESC").Offset(offset(page, pageSize)).Limit(pageSize).Find(&revisions).Error
	return revisions, total, err
}

func (s *Store) RollbackApp(appName string, version int64, actorUserID uint64) (*App, error) {
	var app App
	if err := s.db.Where("app_name = ?", appName).First(&app).Error; err != nil {
		return nil, err
	}
	var revision AppRevision
	if err := s.db.Where("app_name = ? AND version = ?", appName, version).First(&revision).Error; err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		app.Format = revision.Format
		app.Content = revision.Content
		app.Sensitive = revision.Sensitive
		app.Version++
		app.UpdatedAt = time.Now().UTC()
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		return createRevisionAndAudit(tx, &app, actorUserID, fmt.Sprintf("回滚到版本 %d", version), "app.rollback")
	}); err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) CreateAPIKey(name, keyHash, keyPreview string, appNames []string, expiresAt *time.Time, actorUserID uint64) (*APIKey, error) {
	now := time.Now().UTC()
	apiKey := &APIKey{Name: name, KeyHash: keyHash, KeyPreview: keyPreview, Status: domain.StatusActive, ExpiresAt: normalizeTimePtr(expiresAt), CreatedAt: now, UpdatedAt: now}
	return apiKey, s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(apiKey).Error; err != nil {
			return err
		}
		if err := replaceAPIKeyApps(tx, apiKey.ID, appNames); err != nil {
			return err
		}
		return createAudit(tx, "user", &actorUserID, "api_key.create", "api_key", fmt.Sprint(apiKey.ID), "")
	})
}

func (s *Store) ListAPIKeys(page, pageSize int) ([]APIKey, int64, error) {
	db := s.db.Model(&APIKey{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var keys []APIKey
	err := db.Order("id DESC").Offset(offset(page, pageSize)).Limit(pageSize).Find(&keys).Error
	return keys, total, err
}

func (s *Store) ReplaceAPIKeyApps(apiKeyID uint64, appNames []string, actorUserID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := replaceAPIKeyApps(tx, apiKeyID, appNames); err != nil {
			return err
		}
		return createAudit(tx, "user", &actorUserID, "api_key.apps.update", "api_key", fmt.Sprint(apiKeyID), "")
	})
}

func (s *Store) DeleteAPIKey(id uint64, actorUserID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_key_id = ?", id).Delete(&APIKeyApp{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&APIKey{}, id).Error; err != nil {
			return err
		}
		return createAudit(tx, "user", &actorUserID, "api_key.delete", "api_key", fmt.Sprint(id), "")
	})
}

func (s *Store) ListAPIKeyApps(apiKeyID uint64) ([]App, error) {
	var links []APIKeyApp
	if err := s.db.Where("api_key_id = ?", apiKeyID).Find(&links).Error; err != nil {
		return nil, err
	}
	appIDs := make([]uint64, 0, len(links))
	for _, link := range links {
		appIDs = append(appIDs, link.AppID)
	}
	var apps []App
	if len(appIDs) > 0 {
		if err := s.db.Where("id IN ?", appIDs).Order("app_name ASC").Find(&apps).Error; err != nil {
			return nil, err
		}
	}
	return apps, nil
}

func (s *Store) FindActiveAPIKeyByHash(hash string) (*APIKey, error) {
	var apiKey APIKey
	if err := s.db.Where("key_hash = ? AND status = ?", hash, domain.StatusActive).First(&apiKey).Error; err != nil {
		return nil, err
	}
	if apiKey.ExpiresAt != nil && time.Now().UTC().After(*apiKey.ExpiresAt) {
		return nil, gorm.ErrRecordNotFound
	}
	return &apiKey, nil
}

func (s *Store) APIKeyCanAccessApp(apiKeyID, appID uint64) (bool, error) {
	var count int64
	err := s.db.Model(&APIKeyApp{}).Where("api_key_id = ? AND app_id = ?", apiKeyID, appID).Count(&count).Error
	return count > 0, err
}

func (s *Store) TouchAPIKey(id uint64) error {
	now := time.Now().UTC()
	return s.db.Model(&APIKey{}).Where("id = ?", id).Updates(map[string]any{"last_used_at": now, "updated_at": now}).Error
}

func createRevisionAndAudit(tx *gorm.DB, app *App, actorUserID uint64, changeSummary string, action string) error {
	now := time.Now().UTC()
	if err := tx.Create(&AppRevision{
		AppID:           app.ID,
		AppName:         app.AppName,
		Version:         app.Version,
		Format:          app.Format,
		Content:         app.Content,
		Sensitive:       app.Sensitive,
		ChangeSummary:   changeSummary,
		CreatedByUserID: &actorUserID,
		CreatedAt:       now,
	}).Error; err != nil {
		return err
	}
	return createAudit(tx, "user", &actorUserID, action, "app", app.AppName, changeSummary)
}

func createAudit(tx *gorm.DB, actorType string, actorID *uint64, action, resourceType, resourceID, metadata string) error {
	return tx.Create(&AuditLog{
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}).Error
}

func replaceAPIKeyApps(tx *gorm.DB, apiKeyID uint64, appNames []string) error {
	if err := tx.Where("api_key_id = ?", apiKeyID).Delete(&APIKeyApp{}).Error; err != nil {
		return err
	}
	if len(appNames) == 0 {
		return errors.New("至少需要选择一个应用")
	}
	for _, appName := range appNames {
		var app App
		if err := tx.Where("app_name = ?", appName).First(&app).Error; err != nil {
			return err
		}
		if err := tx.Create(&APIKeyApp{APIKeyID: apiKeyID, AppID: app.ID, CreatedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func offset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return (page - 1) * pageSize
}
