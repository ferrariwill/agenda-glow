package service

import (
	"bytes"
	"testing"
)

func TestNewEarlySlotTokenIsOpaqueHashedAndUnique(t *testing.T) {
	tokenA, hashA, err := newEarlySlotToken()
	if err != nil {
		t.Fatal(err)
	}
	tokenB, hashB, err := newEarlySlotToken()
	if err != nil {
		t.Fatal(err)
	}
	if tokenA == "" || tokenB == "" || tokenA == tokenB {
		t.Fatalf("tokens must be non-empty and unique")
	}
	if bytes.Equal([]byte(tokenA), hashA[:]) {
		t.Fatal("plaintext token must not be persisted as its hash")
	}
	if hashEarlySlotToken(tokenA) != hashA {
		t.Fatal("lookup hash must equal persisted SHA-256")
	}
	if hashA == hashB {
		t.Fatal("independent tokens must have distinct hashes")
	}
}

func TestWhatsAppEarlySlotCallbackRequiresScopedOpaqueToken(t *testing.T) {
	valid := WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Action:        WhatsAppActionEarlySlotAccept,
		OfferToken:    "opaque-token",
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid callback rejected: %v", err)
	}

	withoutToken := valid
	withoutToken.OfferToken = ""
	if err := withoutToken.validate(); err == nil {
		t.Fatal("callback without offer_token must be rejected")
	}

	withoutTenant := valid
	withoutTenant.TenantID = ""
	if err := withoutTenant.validate(); err == nil {
		t.Fatal("callback without tenant_id must be rejected")
	}
}
