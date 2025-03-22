package test

import (
	"context"
	"testing"

	"github.com/example/nfsserver/proto"
)

// Test server implementation
type testServer struct {
	api.UnimplementedGreeterServer
}

func (s *testServer) SayHello(ctx context.Context, req *api.HelloRequest) (*api.HelloReply, error) {
	return &api.HelloReply{Message: "Hello, " + req.GetName()}, nil
}

func TestSayHello(t *testing.T) {
	s := &testServer{}
	
	// Test cases
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Basic", "test", "Hello, test"},
		{"Empty", "", "Hello, "},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &api.HelloRequest{Name: tc.input}
			resp, err := s.SayHello(context.Background(), req)
			
			if err != nil {
				t.Errorf("SayHello(%v) error: %v", tc.input, err)
			}
			
			if resp.GetMessage() != tc.expected {
				t.Errorf("SayHello(%v) = %v, expected %v", tc.input, resp.GetMessage(), tc.expected)
			}
		})
	}
}
