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
