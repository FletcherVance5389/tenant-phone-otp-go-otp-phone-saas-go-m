package main

import "context"

type PhoneVerifier interface {
	SendCode(context.Context, string, string, string) error
	VerifyCode(context.Context, string, string, bool) error
	VerifyCaptcha(context.Context, string, string, string) error
}

type PhoneLoginService struct {
	directory *TenantDirectory
	infrai    PhoneVerifier
}

func (s *PhoneLoginService) RequestCode(ctx context.Context, tenantID, phone, widgetRecordID, captchaToken, ip string) error {
	if err := s.directory.AllowLogin(tenantID, phone); err != nil {
		return err
	}
	if err := s.infrai.VerifyCaptcha(ctx, widgetRecordID, captchaToken, ip); err != nil {
		return err
	}
	return s.infrai.SendCode(ctx, phone, "login", "en")
}

func (s *PhoneLoginService) CompleteLogin(ctx context.Context, tenantID, phone, code string) error {
	if err := s.directory.AllowLogin(tenantID, phone); err != nil {
		return err
	}
	return s.infrai.VerifyCode(ctx, phone, code, true)
}
