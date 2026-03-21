package models

import (
	"strings"
	"testing"
)

func TestUser_Validate_Success(t *testing.T) {
	user := &User{Name: "John Doe"}
	if err := user.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUser_Validate_RequiredName(t *testing.T) {
	user := &User{Name: ""}
	err := user.Validate()
	if err == nil {
		t.Error("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' in error message, got: %v", err)
	}
}

func TestUser_Validate_WhitespaceName(t *testing.T) {
	user := &User{Name: "   "}
	err := user.Validate()
	if err == nil {
		t.Error("expected error for whitespace-only name")
	}
}

func TestUser_Validate_NameTooLong(t *testing.T) {
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}

	user := &User{Name: string(longName)}
	err := user.Validate()
	if err == nil {
		t.Error("expected error for name exceeding 255 characters")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected 'exceeds' in error message, got: %v", err)
	}
}

func TestUser_Sanitize(t *testing.T) {
	user := &User{Name: "  Alice Wonderland  "}
	user.Sanitize()

	if user.Name != "Alice Wonderland" {
		t.Errorf("expected sanitized name 'Alice Wonderland', got '%s'", user.Name)
	}
}
