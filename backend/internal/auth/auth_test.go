package auth

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !CheckPassword("secret123", hash) {
		t.Fatal("valid password rejected")
	}
	if CheckPassword("wrong", hash) {
		t.Fatal("invalid password accepted")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	secret := "test-secret"
	token, err := GenerateToken(secret, 42, "user@test.fr", "admin")
	if err != nil {
		t.Fatalf("generate token error: %v", err)
	}
	identity, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("parse token error: %v", err)
	}
	if identity.UserID != 42 || identity.Role != "admin" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if _, err := ParseToken("other-secret", token); err == nil {
		t.Fatal("expected error with wrong secret")
	}
}
