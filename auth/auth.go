package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents a user session
type Session struct {
	Username  string
	ExpiresAt time.Time
}

var (
	// sessions stores active user sessions: token -> Session
	sessions  = make(map[string]Session)
	sessionMu sync.RWMutex

	SessionDuration = 30 * time.Minute
)

// GenerateRandomToken generates a 60-character hex token (30 bytes)
func GenerateRandomToken() (string, error) {
	b := make([]byte, 30)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashSecret hashes a password or token using bcrypt
func HashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifySecret compares a secret with its bcrypt hash
func VerifySecret(secret, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
	return err == nil
}

// CreateSession creates a new session and returns the token
func CreateSession(username string) (string, error) {
	token, err := GenerateRandomToken()
	if err != nil {
		return "", err
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	sessions[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	return token, nil
}

// ValidateSession checks if a session token is valid and returns the username
// It also refreshes the session expiry if valid.
func ValidateSession(token string) (string, bool) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session, ok := sessions[token]
	if !ok {
		return "", false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessions, token)
		return "", false
	}

	// Refresh session
	session.ExpiresAt = time.Now().Add(SessionDuration)
	sessions[token] = session

	return session.Username, true
}

// DeleteSession removes a session
func DeleteSession(token string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	delete(sessions, token)
}

// CleanupSessions removes expired sessions from memory
func CleanupSessions() {
	for {
		time.Sleep(5 * time.Minute)
		sessionMu.Lock()
		now := time.Now()
		for token, session := range sessions {
			if now.After(session.ExpiresAt) {
				delete(sessions, token)
			}
		}
		sessionMu.Unlock()
	}
}

func init() {
	go CleanupSessions()
}
