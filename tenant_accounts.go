package main

import (
	"errors"
	"sync"
)

var (
	errTenantMissing   = errors.New("tenant not found")
	errAccountMissing  = errors.New("account not found")
	errAccountInactive = errors.New("account is not active")
)

type AccountState string

const (
	AccountActive    AccountState = "active"
	AccountSuspended AccountState = "suspended"
)

type Account struct {
	TenantID string       `json:"tenant_id"`
	Phone    string       `json:"phone"`
	State    AccountState `json:"state"`
}

type TenantDirectory struct {
	mu       sync.RWMutex
	tenants  map[string]bool
	accounts map[string]Account
}

func NewTenantDirectory() *TenantDirectory {
	return &TenantDirectory{tenants: make(map[string]bool), accounts: make(map[string]Account)}
}

func accountKey(tenantID, phone string) string { return tenantID + "\x00" + phone }

func (d *TenantDirectory) Onboard(tenantID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tenants[tenantID] = true
}

func (d *TenantDirectory) AddAccount(tenantID, phone string) (Account, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.tenants[tenantID] {
		return Account{}, errTenantMissing
	}
	account := Account{TenantID: tenantID, Phone: phone, State: AccountActive}
	d.accounts[accountKey(tenantID, phone)] = account
	return account, nil
}

func (d *TenantDirectory) SetState(tenantID, phone string, state AccountState) (Account, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := accountKey(tenantID, phone)
	account, ok := d.accounts[key]
	if !ok {
		return Account{}, errAccountMissing
	}
	account.State = state
	d.accounts[key] = account
	return account, nil
}

func (d *TenantDirectory) AllowLogin(tenantID, phone string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	account, ok := d.accounts[accountKey(tenantID, phone)]
	if !ok {
		return errAccountMissing
	}
	if account.State != AccountActive {
		return errAccountInactive
	}
	return nil
}
