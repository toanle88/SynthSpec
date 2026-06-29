package generator

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestComputeSha256(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		if computeSha256("hello") != computeSha256("hello") {
			t.Errorf("sha256 must be deterministic")
		}
	})

	t.Run("known value", func(t *testing.T) {
		h := sha256.New()
		h.Write([]byte("hello"))
		expected := fmt.Sprintf("%x", h.Sum(nil))
		result := computeSha256("hello")
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		h := sha256.New()
		h.Write([]byte(""))
		expected := fmt.Sprintf("%x", h.Sum(nil))
		result := computeSha256("")
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("unicode", func(t *testing.T) {
		h := sha256.New()
		h.Write([]byte("héllo ⚡"))
		expected := fmt.Sprintf("%x", h.Sum(nil))
		result := computeSha256("héllo ⚡")
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("different inputs differ", func(t *testing.T) {
		if computeSha256("abc") == computeSha256("xyz") {
			t.Errorf("different inputs must produce different hashes")
		}
	})
}

func TestComputeSha256_LargeInput(t *testing.T) {
	input := string(make([]byte, 10000))
	h := sha256.New()
	h.Write([]byte(input))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	result := computeSha256(input)
	if result != expected {
		t.Errorf("large input hash mismatch")
	}
}
