package golog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/KernelPryanic/golog"
	"github.com/stretchr/testify/require"
)

// TestWithCensoredSecretFields tests censoring of secret fields in structs.
func TestWithCensoredSecretFields(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.New(false, &buf)

	type SubStruct struct {
		PublicField string `json:"public"`
		SecretField string `json:"secret" secret:"true"`
	}

	type Config struct {
		Username   string     `json:"username"`
		Password   string     `json:"password" secret:"true"`
		APIKey     string     `json:"api_key" secret:"true"`
		SubConfig  *SubStruct `json:"sub_config"`
		PublicData string     `json:"public_data"`
	}

	cfg := Config{
		Username:   "admin",
		Password:   "super_secret_password",
		APIKey:     "sk_test_1234567890",
		PublicData: "this is public",
		SubConfig: &SubStruct{
			PublicField: "visible",
			SecretField: "hidden",
		},
	}

	ctx := golog.WithCensoredSecretFields(logger.With(), "config", cfg)
	logger = ctx.Logger()
	logger.Info().Msg("test message")

	// Parse the log output
	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	// Verify public fields are logged correctly
	require.Equal(t, "admin", logEntry["config.Username"])
	require.Equal(t, "this is public", logEntry["config.PublicData"])
	require.Equal(t, "visible", logEntry["config.SubConfig.PublicField"])

	// Verify secret fields are censored
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["config.Password"])
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["config.APIKey"])
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["config.SubConfig.SecretField"])
}

// TestWithCensoredSecretFieldsNilPointer tests handling of nil pointer fields.
func TestWithCensoredSecretFieldsNilPointer(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.New(false, &buf)

	type SubStruct struct {
		PublicField string `json:"public"`
		SecretField string `json:"secret" secret:"true"`
	}

	type Config struct {
		Username  string     `json:"username"`
		Password  string     `json:"password" secret:"true"`
		SubConfig *SubStruct `json:"sub_config"`
	}

	cfg := Config{
		Username:  "admin",
		Password:  "secret123",
		SubConfig: nil, // nil pointer
	}

	ctx := golog.WithCensoredSecretFields(logger.With(), "config", cfg)
	logger = ctx.Logger()
	logger.Info().Msg("test with nil pointer")

	// Parse the log output
	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	// Verify top-level fields
	require.Equal(t, "admin", logEntry["config.Username"])
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["config.Password"])

	// Verify nil pointer fields are not present
	require.NotContains(t, logEntry, "config.SubConfig.PublicField")
	require.NotContains(t, logEntry, "config.SubConfig.SecretField")
}

// TestWithCensoredSecretFieldsNonStruct tests passing non-struct values.
func TestWithCensoredSecretFieldsNonStruct(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.New(false, &buf)

	// Pass a non-struct value (should be handled gracefully)
	ctx := golog.WithCensoredSecretFields(logger.With(), "value", 42)
	logger = ctx.Logger()
	logger.Info().Msg("test with non-struct")

	// Parse the log output
	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	// Should only have standard fields (time, level, message)
	require.NotContains(t, logEntry, "value")
	require.Equal(t, "test with non-struct", logEntry["message"])
}

// TestWithCensoredSecretFieldsPointerToStruct tests passing a pointer to struct.
func TestWithCensoredSecretFieldsPointerToStruct(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.New(false, &buf)

	type Config struct {
		Username string `json:"username"`
		Password string `json:"password" secret:"true"`
	}

	cfg := &Config{
		Username: "user1",
		Password: "pass123",
	}

	ctx := golog.WithCensoredSecretFields(logger.With(), "config", cfg)
	logger = ctx.Logger()
	logger.Info().Msg("test with pointer")

	// Parse the log output
	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	// Verify fields
	require.Equal(t, "user1", logEntry["config.Username"])
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["config.Password"])
}

// TestWithCensoredSecretFieldsDeepNesting tests deeply nested struct handling.
func TestWithCensoredSecretFieldsDeepNesting(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.New(false, &buf)

	type Level3 struct {
		Data   string `json:"data"`
		Secret string `json:"secret" secret:"true"`
	}

	type Level2 struct {
		Info   string  `json:"info"`
		Level3 *Level3 `json:"level3"`
	}

	type Level1 struct {
		Name   string  `json:"name"`
		Level2 *Level2 `json:"level2"`
		Token  string  `json:"token" secret:"true"`
	}

	cfg := &Level1{
		Name:  "root",
		Token: "secret_token",
		Level2: &Level2{
			Info: "middle",
			Level3: &Level3{
				Data:   "deep_data",
				Secret: "deep_secret",
			},
		},
	}

	ctx := golog.WithCensoredSecretFields(logger.With(), "cfg", cfg)
	logger = ctx.Logger()
	logger.Info().Msg("deep nesting test")

	// Parse the log output
	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	// Verify public fields at all levels
	require.Equal(t, "root", logEntry["cfg.Name"])
	require.Equal(t, "middle", logEntry["cfg.Level2.Info"])
	require.Equal(t, "deep_data", logEntry["cfg.Level2.Level3.Data"])

	// Verify secret fields at all levels are censored
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["cfg.Token"])
	require.Equal(t, golog.CensoredFieldPlaceholder, logEntry["cfg.Level2.Level3.Secret"])
}
