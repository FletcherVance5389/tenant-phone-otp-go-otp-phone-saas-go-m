package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

type server struct {
	directory *TenantDirectory
	login     *PhoneLoginService
}

type phoneInput struct {
	TenantID       string `json:"tenant_id"`
	Phone          string `json:"phone"`
	Code           string `json:"code"`
	WidgetRecordID string `json:"widget_record_id"`
	CaptchaToken   string `json:"captcha_token"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	directory := NewTenantDirectory()
	api := NewInfraiClient(key)
	s := &server{directory: directory, login: &PhoneLoginService{directory: directory, infrai: api}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/tenants/{tenant}", s.onboardTenant)
	mux.HandleFunc("POST /admin/tenants/{tenant}/accounts", s.addAccount)
	mux.HandleFunc("POST /admin/tenants/{tenant}/accounts/suspend", s.suspendAccount)
	mux.HandleFunc("POST /admin/tenants/{tenant}/accounts/reactivate", s.reactivateAccount)
	mux.HandleFunc("POST /login/code", s.requestCode)
	mux.HandleFunc("POST /login/verify", s.verifyCode)
	log.Printf("phone login service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *server) onboardTenant(w http.ResponseWriter, r *http.Request) {
	s.directory.Onboard(r.PathValue("tenant"))
	writeJSON(w, http.StatusCreated, map[string]string{"state": "onboarded"})
}

func (s *server) addAccount(w http.ResponseWriter, r *http.Request) {
	var in phoneInput
	if decode(r, &in) != nil || in.Phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone is required"})
		return
	}
	account, err := s.directory.AddAccount(r.PathValue("tenant"), in.Phone)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func (s *server) suspendAccount(w http.ResponseWriter, r *http.Request) {
	s.changeState(w, r, AccountSuspended)
}

func (s *server) reactivateAccount(w http.ResponseWriter, r *http.Request) {
	s.changeState(w, r, AccountActive)
}

func (s *server) changeState(w http.ResponseWriter, r *http.Request, state AccountState) {
	var in phoneInput
	if decode(r, &in) != nil || in.Phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone is required"})
		return
	}
	account, err := s.directory.SetState(r.PathValue("tenant"), in.Phone, state)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (s *server) requestCode(w http.ResponseWriter, r *http.Request) {
	var in phoneInput
	if decode(r, &in) != nil || in.TenantID == "" || in.Phone == "" || in.WidgetRecordID == "" || in.CaptchaToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id, phone, widget_record_id, and captcha_token are required"})
		return
	}
	err := s.login.RequestCode(r.Context(), in.TenantID, in.Phone, in.WidgetRecordID, in.CaptchaToken, r.RemoteAddr)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"state": "code_sent"})
}

func (s *server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in phoneInput
	if decode(r, &in) != nil || in.TenantID == "" || in.Phone == "" || in.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id, phone, and code are required"})
		return
	}
	err := s.login.CompleteLogin(r.Context(), in.TenantID, in.Phone, in.Code)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"state": "authenticated", "tenant_id": in.TenantID})
}

func writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, errAccountInactive) {
		status = http.StatusForbidden
	} else if errors.Is(err, errTenantMissing) || errors.Is(err, errAccountMissing) {
		status = http.StatusNotFound
	} else {
		var apiErr *InfraiError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			status = apiErr.HTTPStatus
		}
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
