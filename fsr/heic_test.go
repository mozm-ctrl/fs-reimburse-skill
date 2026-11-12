package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHEIC(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sample.heic"))
	if err != nil {
		t.Fatal(err)
	}
	u, err := ToDataURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeDataURL(t, u).Bounds(); b.Dx() == 0 || b.Dx() > 800 {
		t.Fatalf("got %v, want width 1-800", b)
	}
}

func TestEnvCredentials(t *testing.T) {
	for _, k := range []string{"yx", "mm", "FSR_EMAIL", "FSR_PASSWORD"} {
		t.Setenv(k, "")
	}
	if _, _, ok := envCredentials(); ok {
		t.Fatal("expected no credentials")
	}
	if _, _, err := session(); err != errNoEnv {
		t.Fatalf("session without env: got %v, want errNoEnv", err)
	}
	t.Setenv("yx", "a@x")
	t.Setenv("mm", "p")
	if e, _, ok := envCredentials(); !ok || e != "a@x" {
		t.Fatalf("yx/mm: got %q %v", e, ok)
	}
	t.Setenv("FSR_EMAIL", "b@x")
	t.Setenv("FSR_PASSWORD", "q")
	if e, p, ok := envCredentials(); !ok || e != "b@x" || p != "q" {
		t.Fatalf("FSR_* should win: got %q %q %v", e, p, ok)
	}
}
