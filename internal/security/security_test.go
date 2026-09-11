package security

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash, salt, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "" || salt == "" {
		t.Fatal("hash and salt are required")
	}
	if !VerifyPassword("secret", hash, salt) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("wrong", hash, salt) {
		t.Fatal("expected wrong password to fail")
	}
}

func TestAPIKeyHash(t *testing.T) {
	apiKey, err := NewAPIKey()
	if err != nil {
		t.Fatalf("new api key: %v", err)
	}
	if len(apiKey) <= len("mgp_") || apiKey[:4] != "mgp_" {
		t.Fatalf("unexpected api key format: %q", apiKey)
	}
	if HashAPIKey(apiKey) == HashAPIKey(apiKey+"x") {
		t.Fatal("hash should change when api key changes")
	}
}
