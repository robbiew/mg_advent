package validation

import (
	"testing"
	"testing/fstest"
	"time"
)

func TestRequireKey(t *testing.T) {
	// Create a mock filesystem
	mockFS := fstest.MapFS{}

	// Create a validator
	validator := NewValidator(mockFS, "art")

	// Use monkey patching to override time.Now for testing
	// Save the original function
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Test with current year set to 2025
	timeNow = func() time.Time {
		return time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	}

	// Test cases for 2025
	testCases := []struct {
		name     string
		year     int
		expected bool
	}{
		{"Current year (2025) doesn't require key", 2025, false},
		{"Older year (2024) requires key when current year is 2025", 2024, true},
		{"Older year (2023) requires key when current year is 2025", 2023, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.RequireKey(tc.year)
			if result != tc.expected {
				t.Errorf("RequireKey(%d) = %v, expected %v", tc.year, result, tc.expected)
			}
		})
	}

	// Now test with current year set to 2024
	timeNow = func() time.Time {
		return time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	}

	// Additional test cases for 2024
	additionalTestCases := []struct {
		name     string
		year     int
		expected bool
	}{
		{"Current year (2024) doesn't require key", 2024, false},
		{"Older year (2023) doesn't require key when current year is 2024", 2023, false},
	}

	for _, tc := range additionalTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.RequireKey(tc.year)
			if result != tc.expected {
				t.Errorf("RequireKey(%d) = %v, expected %v", tc.year, result, tc.expected)
			}
		})
	}
}
