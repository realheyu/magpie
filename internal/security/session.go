package security

import (
	"sync"
	"time"

	"github.com/bt-smart/btutil/strutil"
)

type Session struct {
	UserID    uint64
	ExpiresAt time.Time
}

type SessionManager struct {
	mu       sync.RWMutex
	ttl      time.Duration
	sessions map[string]Session
}

func NewSessionManager(ttl time.Duration) *SessionManager {
	m := &SessionManager{ttl: ttl, sessions: make(map[string]Session)}
	go m.cleanupLoop(24 * time.Hour)
	return m
}

// cleanupLoop 每天清一次过期会话。Validate 只会在 token 再次被访问时清理，
// 被遗弃的 token 需要靠这里回收，否则长期运行会缓慢累积。
func (m *SessionManager) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UTC()
		m.mu.Lock()
		for token, session := range m.sessions {
			if now.After(session.ExpiresAt) {
				delete(m.sessions, token)
			}
		}
		m.mu.Unlock()
	}
}

func (m *SessionManager) Create(userID uint64) (string, time.Time, error) {
	raw, err := strutil.GenerateRandomString(64, strutil.AllLettersAndDigits)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(m.ttl)
	m.mu.Lock()
	m.sessions[raw] = Session{UserID: userID, ExpiresAt: expiresAt}
	m.mu.Unlock()
	return raw, expiresAt, nil
}

func (m *SessionManager) Validate(token string) (uint64, bool) {
	m.mu.RLock()
	session, ok := m.sessions[token]
	m.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		m.Revoke(token)
		return 0, false
	}
	return session.UserID, true
}

func (m *SessionManager) Revoke(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}
