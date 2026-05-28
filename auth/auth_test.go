package auth

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomTokenReturnsHexToken(t *testing.T) {
	token, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken() error = %v", err)
	}
	if len(token) != 60 {
		t.Fatalf("len(token) = %d, want 60", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}
}

func TestGenerateRandomTokenReturnsUniqueTokens(t *testing.T) {
	first, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken() first error = %v", err)
	}
	second, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken() second error = %v", err)
	}
	if first == second {
		t.Fatal("GenerateRandomToken() returned duplicate tokens")
	}
}

func TestHashSecretAndVerifySecret(t *testing.T) {
	hash, err := HashSecret("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashSecret() error = %v", err)
	}

	if !VerifySecret("correct horse battery staple", hash) {
		t.Fatal("VerifySecret() = false, want true")
	}
	if VerifySecret("wrong secret", hash) {
		t.Fatal("VerifySecret() = true for wrong secret, want false")
	}
}

func TestCreateValidateAndDeleteSession(t *testing.T) {
	token, err := CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	username, ok := ValidateSession(token)
	if !ok {
		t.Fatal("ValidateSession() ok = false, want true")
	}
	if username != "ryan" {
		t.Fatalf("ValidateSession() username = %q, want %q", username, "ryan")
	}

	DeleteSession(token)
	if _, ok := ValidateSession(token); ok {
		t.Fatal("ValidateSession() ok = true after DeleteSession(), want false")
	}
}

func TestValidateSessionDeletesExpiredSession(t *testing.T) {
	originalDuration := SessionDuration
	SessionDuration = -time.Second
	t.Cleanup(func() {
		SessionDuration = originalDuration
	})

	token, err := CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, ok := ValidateSession(token); ok {
		t.Fatal("ValidateSession() ok = true for expired session, want false")
	}

	sessionMu.RLock()
	_, exists := sessions[token]
	sessionMu.RUnlock()
	if exists {
		t.Fatal("expired session still exists after ValidateSession()")
	}
}

func TestHashSecretReturnsErrorForTooLongSecret(t *testing.T) {
	if _, err := HashSecret(strings.Repeat("x", 73)); err == nil {
		t.Fatal("HashSecret() error = nil, want error")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	sessionMu.Lock()
	sessions["expired"] = Session{Username: "old", ExpiresAt: time.Now().Add(-time.Minute)}
	sessions["active"] = Session{Username: "new", ExpiresAt: time.Now().Add(time.Minute)}
	sessionMu.Unlock()

	cleanupExpiredSessions(time.Now())

	sessionMu.RLock()
	_, expiredExists := sessions["expired"]
	_, activeExists := sessions["active"]
	sessionMu.RUnlock()

	if expiredExists {
		t.Fatal("expired session still exists")
	}
	if !activeExists {
		t.Fatal("active session was removed")
	}

	DeleteSession("active")
}
