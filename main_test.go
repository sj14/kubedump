package main

import (
	"reflect"
	"testing"
)

func TestDeleteField(t *testing.T) {
	type args struct {
		object map[string]interface{}
		path   []string
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "nil",
			want: nil,
		},
		{
			name: "not found",
			args: args{
				object: map[string]interface{}{"Hello": "World"},
				path:   []string{"Bye"},
			},
			want: map[string]interface{}{"Hello": "World"},
		},

		{
			name: "happy flat",
			args: args{
				object: map[string]interface{}{"Hello": "World"},
				path:   []string{"Hello"},
			},
			want: map[string]interface{}{},
		},
		{
			name: "happy nested",
			args: args{
				object: map[string]interface{}{"Hello": map[string]interface{}{"My": "World"}},
				path:   []string{"Hello", "My"},
			},
			want: map[string]interface{}{"Hello": map[string]interface{}{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deleteField(tt.args.object, tt.args.path...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deleteField() = %v, want %v", got, tt.want)
			}
		})
	}
}
