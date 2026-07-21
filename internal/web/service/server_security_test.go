package service

import (
	"strings"
	"testing"
)

func TestGetRemoteCertHashBlocksPrivateTargets(t *testing.T) {
	for _, target := range []string{"127.0.0.1:443", "10.0.0.1:443", "[::1]:443"} {
		if _, err := (&ServerService{}).GetRemoteCertHash(target); err == nil {
			t.Fatalf("private certificate target %q was accepted", target)
		}
	}
}

func TestGetRemoteCertHashRejectsInvalidPort(t *testing.T) {
	_, err := (&ServerService{}).GetRemoteCertHash("example.com:70000")
	if err == nil || !strings.Contains(err.Error(), "1-65535") {
		t.Fatalf("invalid port error = %v, want explicit range error", err)
	}
}
