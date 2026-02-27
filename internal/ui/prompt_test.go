package ui

import (
	"errors"
	"testing"
)

func TestSelectOption_WhenNotInteractive_ReturnsErrorOrEmptyResult(t *testing.T) {
	opts := []string{"a", "b"}
	var out string
	err := SelectOption("Choose", opts, &out)
	if err == nil && out != "" {
		t.Errorf("when not interactive: expect err or empty out, got out=%q", out)
	}
	if err != nil && !errors.Is(err, ErrNotInteractive) {
		t.Errorf("when not interactive: expect ErrNotInteractive, got %v", err)
	}
}

func TestConfirm_WhenNotInteractive_ReturnsDefault(t *testing.T) {
	yes, err := Confirm("Sure?", true)
	if err != nil {
		t.Fatalf("Confirm default true: %v", err)
	}
	if !yes {
		t.Error("Confirm(_, true) when not interactive should return true")
	}
	no, err := Confirm("Sure?", false)
	if err != nil {
		t.Fatalf("Confirm default false: %v", err)
	}
	if no {
		t.Error("Confirm(_, false) when not interactive should return false")
	}
}

func TestInput_WhenNotInteractive_ReturnsError(t *testing.T) {
	var s string
	err := Input("Name?", &s)
	if err == nil {
		t.Error("Input when not interactive should return error")
	}
	if err != nil && !errors.Is(err, ErrNotInteractive) {
		t.Errorf("expect ErrNotInteractive, got %v", err)
	}
}

func TestInputWithDefault_WhenNotInteractive_ReturnsDefault(t *testing.T) {
	var s string
	err := InputWithDefault("Name?", "fallback", &s)
	if !errors.Is(err, ErrNotInteractive) {
		t.Errorf("expect ErrNotInteractive, got %v", err)
	}
	if s != "fallback" {
		t.Errorf("result = %q, want %q", s, "fallback")
	}
}

func TestPassword_WhenNotInteractive_ReturnsErrNotInteractive(t *testing.T) {
	var s string
	err := Password("API key:", &s)
	if err == nil {
		t.Fatal("Password when not interactive should return error")
	}
	if !errors.Is(err, ErrNotInteractive) {
		t.Errorf("expect ErrNotInteractive, got %v", err)
	}
}

func TestPassword_WhenNotInteractive_DoesNotModifyResult(t *testing.T) {
	s := "original"
	_ = Password("API key:", &s)
	if s != "original" {
		t.Errorf("Password should not modify result on error; got %q", s)
	}
}
