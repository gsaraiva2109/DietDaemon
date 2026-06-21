package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	phc, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	if !strings.HasPrefix(phc, "$argon2id$v=19$") {
		t.Errorf("PHC should start with $argon2id$v=19$, got: %s", phc)
	}

	t.Run("match", func(t *testing.T) {
		ok, err := Verify("correct horse battery staple", phc)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if !ok {
			t.Error("expected match")
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		ok, err := Verify("wrong password", phc)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if ok {
			t.Error("expected mismatch")
		}
	})
}

func TestVerifyTamperedPHC(t *testing.T) {
	phc, err := Hash("a good password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	// Tamper with the hash part. We target the second-to-last character
	// rather than the last: a 32-byte hash encodes to 43 base64 chars and
	// the final character contributes only 4 bits (2 are padding), so 4
	// chars in the same 4-bit group decode identically. The 42nd char has
	// all 6 bits significant — flipping it always changes the hash.
	b := []byte(phc)
	target := b[len(b)-2]
	if target == 'X' {
		b[len(b)-2] = 'Y'
	} else {
		b[len(b)-2] = 'X'
	}
	tampered := string(b)
	ok, err := Verify("a good password", tampered)
	if err != nil {
		t.Fatalf("Verify tampered: %v", err)
	}
	if ok {
		t.Error("tampered PHC should not verify")
	}
}

func TestVerifyMalformedPHC(t *testing.T) {
	ok, err := Verify("anything", "not-a-phc")
	if err != nil {
		t.Fatalf("Verify malformed should not error: %v", err)
	}
	if ok {
		t.Error("malformed PHC should not verify")
	}
}

func TestHashLengthGuards(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		_, err := Hash("short")
		if err != ErrPasswordTooShort {
			t.Errorf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("minimum ok", func(t *testing.T) {
		_, err := Hash("12345678")
		if err != nil {
			t.Errorf("8 chars should be ok: %v", err)
		}
	})

	t.Run("too long", func(t *testing.T) {
		long := make([]byte, 129)
		for i := range long {
			long[i] = 'a'
		}
		_, err := Hash(string(long))
		if err != ErrPasswordTooLong {
			t.Errorf("expected ErrPasswordTooLong, got %v", err)
		}
	})
}

func TestHashDeterministic(t *testing.T) {
	// Two hashes of the same password should produce different PHCs (random salt).
	phc1, _ := Hash("test password 123")
	phc2, _ := Hash("test password 123")
	if phc1 == phc2 {
		t.Error("two hashes of same password should differ (random salt)")
	}

	// Both should verify.
	if ok, _ := Verify("test password 123", phc1); !ok {
		t.Error("phc1 should verify")
	}
	if ok, _ := Verify("test password 123", phc2); !ok {
		t.Error("phc2 should verify")
	}
}

func TestParsePHC(t *testing.T) {
	phc, err := Hash("parse test password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	mem, time, threads, salt, hash, err := parsePHC(phc)
	if err != nil {
		t.Fatalf("parsePHC: %v", err)
	}

	if mem != 65536 {
		t.Errorf("memory = %d, want 65536", mem)
	}
	if time != 3 {
		t.Errorf("time = %d, want 3", time)
	}
	if threads != 4 {
		t.Errorf("threads = %d, want 4", threads)
	}
	if len(salt) != 16 {
		t.Errorf("salt len = %d, want 16", len(salt))
	}
	if len(hash) != 32 {
		t.Errorf("hash len = %d, want 32", len(hash))
	}
}
