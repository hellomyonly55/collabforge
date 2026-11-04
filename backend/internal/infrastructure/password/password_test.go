package password

import (
	"strings"
	"testing"
)

func TestBcryptPasswordBoundaries(t *testing.T) {
	for _, value := range []string{"short", strings.Repeat("a", 73), strings.Repeat("汉", 25)} {
		if Validate(value) == nil {
			t.Fatal("expected invalid password")
		}
	}
	for _, value := range []string{"password123", strings.Repeat("汉", 24)} {
		hash, err := Hash(value)
		if err != nil {
			t.Fatal(err)
		}
		if hash == value || !Matches(hash, value) || Matches(hash, "wrong-password") {
			t.Fatal("password hashing or verification failed")
		}
	}
	if Matches("invalid-hash", "password123") {
		t.Fatal("malformed hash must fail")
	}
}
