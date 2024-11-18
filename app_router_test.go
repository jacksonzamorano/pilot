package pilot

import (
	"reflect"
	"testing"
)

func TestPathListFromString(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty",
			path: "/",
			want: []string{""},
		},
		{
			name: "single",
			path: "/hello",
			want: []string{"hello"},
		},
		{
			name: "multiple with slash",
			path: "/hello/world/test/",
			want: []string{"hello", "world", "test"},
		},
		{
			name: "multiple",
			path: "/hello/world/test",
			want: []string{"hello", "world", "test"},
		},
		{
			name: "identical slashes",
			path: "/hello/test/test",
			want: []string{"hello", "test", "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathListFromString(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PathListFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
