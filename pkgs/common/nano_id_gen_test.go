package common

import (
	"errors"
	"testing"
)

func TestGenNanoID_Length(t *testing.T) {
	id, err := GenNanoID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 10 {
		t.Errorf("GenNanoID len = %d, want 10", len(id))
	}
}

func TestTryGenNanoID_Valid(t *testing.T) {
	id, err := TryGenNanoIDFromAlphaNumber(5, 3, func(string) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 5 {
		t.Errorf("len = %d, want 5", len(id))
	}
}

func TestTryGenNanoID_ExhaustAttempts(t *testing.T) {
	called := 0
	_, err := TryGenNanoIDFromAlphaNumber(5, 3, func(string) (bool, error) {
		called++
		return false, nil
	})
	if err == nil {
		t.Fatal("expected exhaustion error")
	}
	if called != 3 {
		t.Errorf("validator called %d times, want 3", called)
	}
}

func TestTryGenNanoID_ValidatorError(t *testing.T) {
	custom := errors.New("validator failed")
	_, err := TryGenNanoIDFromAlphaNumber(5, 3, func(string) (bool, error) {
		return false, custom
	})
	if !errors.Is(err, custom) {
		t.Fatalf("want %v, got %v", custom, err)
	}
}

// tryCount=-1、0、1 均应返回有效 ID（统一当作至少尝试一次）
func TestTryGenNanoID_NegativeAndZeroTryCountReturnValidID(t *testing.T) {
	tests := []struct {
		name     string
		tryCount int
	}{
		{"tryCount=-1", -1},
		{"tryCount=0", 0},
		{"tryCount=1", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := TryGenNanoIDFromAlphaNumber(6, tt.tryCount, func(string) (bool, error) {
				return true, nil
			})
			if err != nil {
				t.Fatalf("tryCount=%d: unexpected error: %v", tt.tryCount, err)
			}
			if len(id) != 6 {
				t.Errorf("tryCount=%d: len = %d, want 6", tt.tryCount, len(id))
			}
		})
	}
}
