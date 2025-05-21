package queue

import (
	"strconv"
	"testing"
)

func TestPushResultValues(t *testing.T) {
	t.Parallel()

	// Test that the constants are distinct
	results := []PushResult{PushFailed, PushSuccess, PushFull}
	for i, r1 := range results {
		for j, r2 := range results {
			if i != j && r1 == r2 {
				t.Errorf("PushResult constants are not distinct: %d == %d", r1, r2)
			}
		}
	}
}

func TestPopResultValues(t *testing.T) {
	t.Parallel()
	// Test that the constants are distinct
	results := []PopResult{PopFailed, PopSuccess, PopEmpty}
	for i, r1 := range results {
		for j, r2 := range results {
			if i != j && r1 == r2 {
				t.Errorf("PopResult constants are not distinct: %d == %d", r1, r2)
			}
		}
	}
}

func TestPushResultString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result PushResult
		want   string
	}{
		{PushFailed, "PushFailed"},
		{PushSuccess, "PushSuccess"},
		{PushFull, "PushFull"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.result.String()
			if got != tt.want {
				t.Errorf("PushResult(%d).String() = %q, want %q", tt.result, got, tt.want)
			}
		})
	}
}

func TestPopResultString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result PopResult
		want   string
	}{
		{PopFailed, "PopFailed"},
		{PopSuccess, "PopSuccess"},
		{PopEmpty, "PopEmpty"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.result.String()
			if got != tt.want {
				t.Errorf("PopResult(%d).String() = %q, want %q", tt.result, got, tt.want)
			}
		})
	}
}

func TestPushResultIsSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result PushResult
		want   bool
	}{
		{PushFailed, false},
		{PushSuccess, true},
		{PushFull, false},
	}

	for _, tt := range tests {
		t.Run(tt.result.String(), func(t *testing.T) {
			got := tt.result.IsSuccess()
			if got != tt.want {
				t.Errorf("PushResult(%d).IsSuccess() = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

func TestPopResultIsSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result PopResult
		want   bool
	}{
		{PopFailed, false},
		{PopSuccess, true},
		{PopEmpty, false},
	}

	for _, tt := range tests {
		t.Run(tt.result.String(), func(t *testing.T) {
			got := tt.result.IsSuccess()
			if got != tt.want {
				t.Errorf("PopResult(%d).IsSuccess() = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

// Test type casting between result types and int
func TestResultTypeCasting(t *testing.T) {
	t.Parallel()

	// Test PushResult casting
	pushResults := []PushResult{PushFailed, PushSuccess, PushFull}
	for _, r := range pushResults {
		intValue := int(r)
		backToPushResult := PushResult(intValue)
		if backToPushResult != r {
			t.Errorf("Type casting failed for PushResult: %v != %v", backToPushResult, r)
		}
	}

	// Test PopResult casting
	popResults := []PopResult{PopFailed, PopSuccess, PopEmpty}
	for _, r := range popResults {
		intValue := int(r)
		backToPopResult := PopResult(intValue)
		if backToPopResult != r {
			t.Errorf("Type casting failed for PopResult: %v != %v", backToPopResult, r)
		}
	}
}

// TestPushResultStringPanic verifies that the String method panics when given an unknown PushResult value
func TestPushResultStringPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("PushResult(999).String() did not panic as expected")
		} else if r != "unknown PushResult" {
			t.Errorf("PushResult(999).String() panicked with unexpected message: %v", r)
		}
	}()

	// This should panic
	invalidResult := PushResult(999)
	_ = invalidResult.String()
}

// TestPopResultStringPanic verifies that the String method panics when given an unknown PopResult value
func TestPopResultStringPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("PopResult(999).String() did not panic as expected")
		} else if r != "unknown PopResult" {
			t.Errorf("PopResult(999).String() panicked with unexpected message: %v", r)
		}
	}()

	// This should panic
	invalidResult := PopResult(999)
	_ = invalidResult.String()
}

// TestPushResultStringMultipleValues tests multiple invalid values for panic consistency
func TestPushResultStringMultipleValues(t *testing.T) {
	t.Parallel()

	invalidValues := []PushResult{-999, -100, -2, 2, 3, 100, 999}

	for _, val := range invalidValues {
		testValue := val // Capture for closure
		t.Run("Value-"+strconv.Itoa(int(testValue)), func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("PushResult(%d).String() did not panic as expected", testValue)
				} else if r != "unknown PushResult" {
					t.Errorf("PushResult(%d).String() panicked with unexpected message: %v", testValue, r)
				}
			}()

			_ = testValue.String()
		})
	}
}

// TestPopResultStringMultipleValues tests multiple invalid values for panic consistency
func TestPopResultStringMultipleValues(t *testing.T) {
	t.Parallel()

	invalidValues := []PopResult{-999, -100, -2, 2, 3, 100, 999}

	for _, val := range invalidValues {
		testValue := val // Capture for closure
		t.Run("Value-"+strconv.Itoa(int(testValue)), func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("PopResult(%d).String() did not panic as expected", testValue)
				} else if r != "unknown PopResult" {
					t.Errorf("PopResult(%d).String() panicked with unexpected message: %v", testValue, r)
				}
			}()

			_ = testValue.String()
		})
	}
}
