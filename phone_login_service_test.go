package main

import (
	"context"
	"errors"
	"testing"
)

type fakeVerifier struct {
	verifyCalls int
}

func (f *fakeVerifier) SendCode(context.Context, string, string, string) error      { return nil }
func (f *fakeVerifier) VerifyCaptcha(context.Context, string, string, string) error { return nil }
func (f *fakeVerifier) VerifyCode(context.Context, string, string, bool) error {
	f.verifyCalls++
	return nil
}

func TestCompleteLoginHonorsAccountLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		state     AccountState
		wantErr   error
		wantCalls int
	}{
		{name: "active account verifies OTP", state: AccountActive, wantCalls: 1},
		{name: "suspended account is rejected before OTP verification", state: AccountSuspended, wantErr: errAccountInactive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := NewTenantDirectory()
			directory.Onboard("acme")
			if _, err := directory.AddAccount("acme", "+12025550123"); err != nil {
				t.Fatal(err)
			}
			if _, err := directory.SetState("acme", "+12025550123", tt.state); err != nil {
				t.Fatal(err)
			}
			fake := &fakeVerifier{}
			service := &PhoneLoginService{directory: directory, infrai: fake}
			err := service.CompleteLogin(context.Background(), "acme", "+12025550123", "123456")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CompleteLogin() error = %v, want %v", err, tt.wantErr)
			}
			if fake.verifyCalls != tt.wantCalls {
				t.Fatalf("VerifyCode calls = %d, want %d", fake.verifyCalls, tt.wantCalls)
			}
		})
	}
}
