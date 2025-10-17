package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		envValue string
		setEnv   bool
		expected string
	}{
		{"existing_var", "TEST_STRING", "default", "production", true, "production"},
		{"missing_var", "MISSING_KEY", "default", "", false, "default"},
		{"empty_var", "EMPTY_KEY", "default", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetString(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback int
		envValue string
		setEnv   bool
		expected int
	}{
		{"valid_int", "TEST_INT", 10, "42", true, 42},
		{"invalid_int", "INVALID_INT", 10, "not_a_number", true, 10},
		{"missing_int", "MISSING_INT", 10, "", false, 10},
		{"negative_int", "NEG_INT", 10, "-5", true, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetInt(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback bool
		envValue string
		setEnv   bool
		expected bool
	}{
		{"true_string", "TEST_BOOL", false, "true", true, true},
		{"false_string", "TEST_BOOL", true, "false", true, false},
		{"one_as_true", "TEST_BOOL", false, "1", true, true},
		{"zero_as_false", "TEST_BOOL", true, "0", true, false},
		{"invalid_bool", "INVALID_BOOL", false, "maybe", true, false},
		{"missing_bool", "MISSING_BOOL", true, "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetBool(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}