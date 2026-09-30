package env_test

import (
	"context"
	"os"
	"testing"

	"github.com/Denio1337/goconf/source/env"
)

func TestEnvSource(t *testing.T) {
	customEnv := []string{
		"APP_NAME=MyApp",
		"APP_SERVER__PORT=8080",
		"APP_SERVER__HOST=127.0.0.1",
		"OTHER_VAR=ignore",
		"INVALID_FORMAT",
		"=missing_key",
	}

	src := env.New(
		env.WithEnviron(customEnv),
		env.WithPrefix("APP_"),
		env.WithStripPrefix(true),
	)

	if src.Name() != "env:prefix=APP_" {
		t.Errorf("expected name 'env:prefix=APP_', got %q", src.Name())
	}

	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	// Should contain stripped keys
	if data["NAME"] != "MyApp" {
		t.Errorf("expected NAME=MyApp, got %v", data["NAME"])
	}
	if data["SERVER__PORT"] != "8080" {
		t.Errorf("expected SERVER__PORT=8080, got %v", data["SERVER__PORT"])
	}
	if data["SERVER.PORT"] != "8080" {
		t.Errorf("expected SERVER.PORT=8080, got %v", data["SERVER.PORT"])
	}

	// Should also preserve original prefixed keys
	if data["APP_NAME"] != "MyApp" {
		t.Errorf("expected APP_NAME=MyApp, got %v", data["APP_NAME"])
	}
	if data["APP_SERVER.PORT"] != "8080" {
		t.Errorf("expected APP_SERVER.PORT=8080, got %v", data["APP_SERVER.PORT"])
	}

	// Should not contain filtered keys
	if _, ok := data["OTHER_VAR"]; ok {
		t.Errorf("expected OTHER_VAR to be filtered out")
	}
}

func TestEnvSourceNoPrefix(t *testing.T) {
	customEnv := []string{
		"PORT=3000",
		"DB__HOST=localhost",
	}

	src := env.New(env.WithEnviron(customEnv))
	if src.Name() != "env" {
		t.Errorf("expected name 'env', got %q", src.Name())
	}

	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if data["PORT"] != "3000" {
		t.Errorf("expected PORT=3000, got %v", data["PORT"])
	}
	if data["DB.HOST"] != "localhost" {
		t.Errorf("expected DB.HOST=localhost, got %v", data["DB.HOST"])
	}
}

func TestEnvSourceRealEnviron(t *testing.T) {
	testKey := "GOCONF_TEST_ENV_VAR"
	testVal := "hello_world"
	os.Setenv(testKey, testVal)
	defer os.Unsetenv(testKey)

	src := env.New(env.WithPrefix("GOCONF_TEST_"))
	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if data["ENV_VAR"] != testVal {
		t.Errorf("expected stripped key ENV_VAR=%s, got %v", testVal, data["ENV_VAR"])
	}
	if data[testKey] != testVal {
		t.Errorf("expected full key %s=%s, got %v", testKey, testVal, data[testKey])
	}
}
