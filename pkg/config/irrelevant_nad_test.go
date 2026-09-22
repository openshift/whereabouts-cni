package config

import (
	"fmt"
	"testing"
)

func TestIsIrrelevantNADError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "missing ipam", err: NewMissingIPAMError(), want: true},
		{name: "invalid plugin", err: NewInvalidPluginError("host-local"), want: true},
		{name: "nil error", err: nil, want: false},
		{name: "other error", err: fmt.Errorf("storage engine"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsIrrelevantNADError(tt.err); got != tt.want {
				t.Fatalf("IsIrrelevantNADError() = %v, want %v", got, tt.want)
			}
		})
	}
}
