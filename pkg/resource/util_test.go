package resource

import "testing"

func TestResourceTypePrefixed(t *testing.T) {
	type args struct {
		rType string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrefixResourceType(tt.args.rType); got != tt.want {
				t.Errorf("PrefixResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}
