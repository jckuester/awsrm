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
		{
			name: "with prefix",
			args: args{
				rType: "aws_instance",
			},
			want: "aws_instance",
		},
		{
			name: "without prefix",
			args: args{
				rType: "instance",
			},
			want: "aws_instance",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrefixResourceType(tt.args.rType); got != tt.want {
				t.Errorf("PrefixResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}
