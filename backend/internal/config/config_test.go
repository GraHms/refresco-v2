package config

import (
	"strings"
	"testing"
)

func setEnv(t *testing.T, kv map[string]string) {
	for _, k := range []string{"APP_ENV", "JWT_SECRET", "APP_SECRET", "MOCK_WEBHOOK_SECRET", "COOKIE_SECURE", "PAYMENTS_ALLOW_MOCK_IN_PRODUCTION", "PAYMENTS_ALLOW_SIMULATE"} {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

var strong = strings.Repeat("x", 40)

func TestMissingEnvMeansProduction(t *testing.T) {
	setEnv(t, nil)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("unset APP_ENV must be treated as production and require secrets, got %v", err)
	}
	setEnv(t, map[string]string{"APP_ENV": "staging"})
	if _, err := Load(); err == nil {
		t.Fatal("unknown APP_ENV accepted")
	}
}

func TestMockRefusedInProduction(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": "production", "JWT_SECRET": strong, "APP_SECRET": strong, "MOCK_WEBHOOK_SECRET": strong})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "mock") {
		t.Fatalf("mock gateway accepted in production: %v", err)
	}
	t.Setenv("PAYMENTS_ALLOW_MOCK_IN_PRODUCTION", "true")
	c, err := Load()
	if err != nil || !c.Sandbox || c.AllowSimulate {
		t.Fatalf("explicit staging override: %+v %v", c, err)
	}
	t.Setenv("PAYMENTS_ALLOW_SIMULATE", "true")
	if _, err := Load(); err == nil {
		t.Fatal("simulate accepted in production")
	}
}

func TestDevSecretsAreRandom(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": "development"})
	a, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Load()
	if string(a.JWTSecret) == string(b.JWTSecret) || len(a.JWTSecret) < 32 || strings.Contains(string(a.JWTSecret), "dev-only") {
		t.Fatal("development secrets must be random per process")
	}
}

func TestMtocaTakesTheWalletsAndWantsAPrivateAddress(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	setEnv(t, map[string]string{"APP_ENV": "production", "JWT_SECRET": secret, "APP_SECRET": secret, "PAYMENTS_PROVIDER": "mtoca", "MTOCA_URL": "", "DATABASE_URL": "postgres://u:p@db/x", "PUBLIC_URL": "https://pagarefresco.com"})
	if _, err := Load(); err == nil {
		t.Fatal("no address must refuse to start")
	}
	t.Setenv("MTOCA_URL", "http://mtoca:8080")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ComingSoon.Emola || c.ComingSoon.Bim || !c.ComingSoon.Card || c.ComingSoon.Mpesa || c.Sandbox {
		t.Fatalf("coming soon %+v, sandbox %v", c.ComingSoon, c.Sandbox)
	}
	for raw, want := range map[string]bool{"http://mtoca:8080": true, "http://172.30.5.2:8080": true, "https://api.example.com": true, "http://api.example.com": false, "http://1.1.1.1:8080": false} {
		if got := checkMtocaURL(raw, true) == nil; got != want {
			t.Errorf("%s: allowed %v", raw, got)
		}
	}
}

func TestMpesaNeedsItsKeysAndKeepsTheRestComingSoon(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": "production", "JWT_SECRET": strong, "APP_SECRET": strong, "COOKIE_SECURE": "true", "PAYMENTS_PROVIDER": "mpesa"})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MPESA_API_KEY") {
		t.Fatalf("started without keys: %v", err)
	}
	t.Setenv("MPESA_API_KEY", "k")
	t.Setenv("MPESA_PUBLIC_KEY", "p")
	t.Setenv("MPESA_SERVICE_PROVIDER_CODE", "171717")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Sandbox || c.AllowSimulate || c.MpesaBaseURL != "https://api.vm.co.mz" || c.MpesaC2BPort != "18352" {
		t.Fatalf("%+v", c)
	}
	if !c.ComingSoon.Emola || !c.ComingSoon.Card || c.ComingSoon.Mpesa {
		t.Fatalf("coming soon: %+v", c.ComingSoon)
	}
}
