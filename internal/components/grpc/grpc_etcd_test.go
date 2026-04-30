package grpc

import "testing"

func TestEtcdTargetUsesWatchedServicePath(t *testing.T) {
	got := etcdTarget("etcd", "grpc_services", "Hello")
	want := "etcd:///grpc_services/Hello"
	if got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
}
