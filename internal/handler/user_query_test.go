package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListUserQuery_Normalize(t *testing.T) {
	tests := []struct {
		name          string
		input         ListUsersQuery
		expectedPage  int
		expectedLimit int
	}{
		{
			name:          "zero values get defaults",
			input:         ListUsersQuery{Page: 0, Limit: 0},
			expectedPage:  defaultPage,
			expectedLimit: defaultLimit,
		},
		{
			name:          "negative values get defaults",
			input:         ListUsersQuery{Page: -5, Limit: -10},
			expectedPage:  defaultPage,
			expectedLimit: defaultLimit,
		},
		{
			name:          "valid values pass through unchanged",
			input:         ListUsersQuery{Page: 3, Limit: 25},
			expectedPage:  3,
			expectedLimit: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.input
			query.Normalize()

			assert.Equal(t, tt.expectedPage, query.Page)
			assert.Equal(t, tt.expectedLimit, query.Limit)
		})
	}
}
