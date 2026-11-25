//go:build unit

package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyCodeKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "normal_email",
			email:    "user@example.com",
			expected: "notify_verify:user@example.com",
		},
		{
			name:     "empty_email",
			email:    "",
			expected: "notify_verify:",
		},
		{
			name:     "email_with_plus",
			email:    "user+tag@example.com",
			expected: "notify_verify:user+tag@example.com",
		},
		{
			name:     "email_with_special_chars",
			email:    "user.name+tag@sub.domain.com",
			expected: "notify_verify:user.name+tag@sub.domain.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := notifyVerifyKey(tc.email)
			require.Equal(t, tc.expected, got)
		})
	}
}
