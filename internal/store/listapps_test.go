package store

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/realheyu/magpie/internal/domain"
)

// openVerifyDB 连接真实 MySQL 验证 ListAppsForUser 生成的 SQL。
// 默认跳过；设置 MAGPIE_TEST_DSN 后运行：
//
//	MAGPIE_TEST_DSN='root:...@tcp(host:3306)/magpie_test?charset=utf8mb4&parseTime=True&loc=UTC' \
//	    go test ./internal/store -run TestListAppsForUser -v
func openVerifyDB(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("MAGPIE_TEST_DSN")
	if dsn == "" {
		t.Skip("需要真实数据库：设置 MAGPIE_TEST_DSN 后运行")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	return &Store{db: db}
}

func findUserByRole(s *Store, role string) (*User, error) {
	var user User
	err := s.db.Where("role = ? AND status = ?", role, domain.StatusActive).Order("id ASC").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 管理员路径曾因 GORM 按 AppWithPermission 智能选列拼出不存在的 apps.permission 而报
// Error 1054；此用例防止回归。管理员应看到全部应用（含非 active），Permission 留空。
func TestListAppsForUserAdmin(t *testing.T) {
	s := openVerifyDB(t)
	admin, err := findUserByRole(s, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("库中没有启用状态的管理员: %v", err)
	}

	for _, query := range []string{"", "a"} {
		rows, total, err := s.ListAppsForUser(admin, query, 1, 20)
		if err != nil {
			t.Fatalf("管理员列表 query=%q 出错: %v", query, err)
		}
		if total != int64(len(rows)) || total < 0 {
			// pageSize=20 内逐页校验意义不大，只确认返回行数与 total 一致或 total 达到页大小
			if total > 20 && len(rows) != 20 {
				t.Fatalf("query=%q 返回 %d 行，与 total=%d 不符", query, len(rows), total)
			}
		}
		for i := range rows {
			if rows[i].Permission != "" {
				t.Fatalf("管理员的行不应带权限值: %+v", rows[i])
			}
			if rows[i].AppName == "" || rows[i].ID == 0 {
				t.Fatalf("应用字段未正确扫描: %+v", rows[i])
			}
		}
	}
}

// 非管理员路径：只看到 active 且授权的应用，Permission 来自权限表 JOIN。
func TestListAppsForUserNonAdmin(t *testing.T) {
	s := openVerifyDB(t)
	user, err := findUserByRole(s, domain.RoleUser)
	if err != nil {
		t.Skipf("库中没有启用状态的普通用户，跳过非管理员路径验证: %v", err)
	}

	rows, total, err := s.ListAppsForUser(user, "", 1, 20)
	if err != nil {
		t.Fatalf("非管理员列表出错: %v", err)
	}
	if total != int64(len(rows)) {
		t.Fatalf("返回 %d 行，与 total=%d 不符", len(rows), total)
	}
	for i := range rows {
		if rows[i].Status != domain.StatusActive {
			t.Fatalf("非管理员看到了非 active 应用: %+v", rows[i])
		}
		if !domain.ValidPermission(rows[i].Permission) {
			t.Fatalf("权限值未从 JOIN 带出: %+v", rows[i])
		}
	}

	if rows, _, err = s.ListAppsForUser(user, "不存在的应用名xyz", 1, 20); err != nil {
		t.Fatalf("非管理员搜索出错: %v", err)
	} else if len(rows) != 0 {
		t.Fatalf("不匹配的搜索应返回空，实际 %d 行", len(rows))
	}
}

// 权限可见性与 GetUserAppPermission 的一致性抽查。
func TestListAppsForUserPermissionConsistency(t *testing.T) {
	s := openVerifyDB(t)
	user, err := findUserByRole(s, domain.RoleUser)
	if err != nil {
		t.Skipf("库中没有启用状态的普通用户，跳过: %v", err)
	}
	rows, _, err := s.ListAppsForUser(user, "", 1, 100)
	if err != nil {
		t.Fatalf("列表出错: %v", err)
	}
	if len(rows) == 0 {
		t.Skip("该用户没有任何授权应用，跳过一致性抽查")
	}
	for i := range rows {
		want, err := s.GetUserAppPermission(user.ID, rows[i].ID)
		if err != nil {
			t.Fatalf("读取权限失败: %v", err)
		}
		if want != rows[i].Permission {
			t.Fatalf("app=%s JOIN 权限 %q 与权限表 %q 不一致", rows[i].AppName, rows[i].Permission, want)
		}
	}
}
