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

func TestTryGenNanoID_ZeroTryCountTreatedAsOne(t *testing.T) {
	called := 0
	id, err := TryGenNanoIDFromAlphaNumber(3, 0, func(string) (bool, error) {
		called++
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 3 {
		t.Errorf("len = %d, want 3", len(id))
	}
	if called != 1 {
		t.Errorf("validator called %d times, want 1", called)
	}
}

func TestTryGenNanoID_NegativeTryCountPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative tryCount")
		}
	}()
	_, _ = TryGenNanoIDFromAlphaNumber(3, -1, func(string) (bool, error) {
		return true, nil
	})
}
