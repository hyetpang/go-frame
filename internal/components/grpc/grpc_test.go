package grpc

import (
	"testing"
	"time"
)

func TestGracefulStopWithTimeoutFallsBackToStop(t *testing.T) {
	stopped := make(chan struct{})
	forced := make(chan struct{}, 1)

	gracefulStopWithTimeout(func() {
		<-stopped
	}, func() {
		forced <- struct{}{}
	}, time.Millisecond)

	select {
	case <-forced:
	case <-time.After(time.Second):
		t.Fatal("expected force stop after timeout")
	}
	close(stopped)
}
