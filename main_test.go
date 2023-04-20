package main

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestSkipResource(t *testing.T) {
	type args struct {
		res             metav1.APIResource
		wantResources   []string
		ignoreResources []string
	}
	tests := []struct {
		name string
		args args
		skip bool
	}{
		{
			name: "empty",
			skip: true,
		},
		{
			name: "verb other",
			args: args{
				res: metav1.APIResource{Verbs: metav1.Verbs{"other"}},
			},
			skip: true,
		},
		{
			name: "verb get",
			args: args{
				res: metav1.APIResource{Verbs: metav1.Verbs{"get"}},
			},
			skip: false,
		},
		{
			name: "subresource",
			args: args{
				res: metav1.APIResource{Name: "resource/subresource", Verbs: metav1.Verbs{"get"}},
			},
			skip: true,
		},

		{
			name: "empty string want/ignore",
			args: args{
				res: metav1.APIResource{
					Name:  "myresource",
					Verbs: metav1.Verbs{"get"},
				},
				wantResources:   []string{""},
				ignoreResources: []string{""},
			},
			skip: false,
		},
		{
			name: "want resource match",
			args: args{
				res: metav1.APIResource{
					Name:  "myresource",
					Verbs: metav1.Verbs{"get"},
				},
				wantResources: []string{"myresource"},
			},
			skip: false,
		},
		{
			name: "want resource don't match",
			args: args{
				res: metav1.APIResource{
					Name:  "not-myresource",
					Verbs: metav1.Verbs{"get"},
				},
				wantResources: []string{"myresource"},
			},
			skip: true,
		},
		{
			name: "ignore resource match",
			args: args{
				res: metav1.APIResource{
					Name:  "myresource",
					Verbs: metav1.Verbs{"get"},
				},
				ignoreResources: []string{"myresource"},
			},
			skip: true,
		},
		{
			name: "ignore resource don't match",
			args: args{
				res: metav1.APIResource{
					Name:  "not-myresource",
					Verbs: metav1.Verbs{"get"},
				},
				ignoreResources: []string{"myresource"},
			},
			skip: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipResource(tt.args.res, tt.args.wantResources, tt.args.ignoreResources); got != tt.skip {
				t.Errorf("ignoreResource() = %v, want %v", got, tt.skip)
			}
		})
	}
}

func TestSkipItem(t *testing.T) {
	type args struct {
		item             unstructured.Unstructured
		namespaced       bool
		clusterscoped    bool
		wantNamespaces   []string
		ignoreNamespaces []string
	}

	namespacedTestItem := unstructured.Unstructured{}
	namespacedTestItem.SetNamespace("mynamespace")

	tests := []struct {
		name string
		args args
		skip bool
	}{
		{
			name: "empty",
			skip: true,
		},
		{
			name: "clusterscoped happy",
			args: args{
				clusterscoped: true,
			},
			skip: false,
		},
		{
			name: "clusterscoped fail",
			args: args{
				item:          namespacedTestItem,
				clusterscoped: true,
			},
			skip: true,
		},
		{
			name: "namespaced fail",
			args: args{
				namespaced: true,
			},
			skip: true,
		},
		{
			name: "namespaced happy",
			args: args{
				item:       namespacedTestItem,
				namespaced: true,
			},
			skip: false,
		},
		{
			name: "want namespace happy",
			args: args{
				item:           namespacedTestItem,
				namespaced:     true,
				wantNamespaces: []string{namespacedTestItem.GetNamespace()},
			},
			skip: false,
		},
		{
			name: "want namespace fail",
			args: args{
				item:           namespacedTestItem,
				namespaced:     true,
				wantNamespaces: []string{"fail-namespace"},
			},
			skip: true,
		},
		{
			name: "ignore namespaces don't match",
			args: args{
				item:             namespacedTestItem,
				namespaced:       true,
				ignoreNamespaces: []string{"other-namespace"},
			},
			skip: false,
		},
		{
			name: "ignore namespaces match",
			args: args{
				item:             namespacedTestItem,
				namespaced:       true,
				ignoreNamespaces: []string{namespacedTestItem.GetNamespace()},
			},
			skip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipItem(tt.args.item, tt.args.namespaced, tt.args.clusterscoped, tt.args.wantNamespaces, tt.args.ignoreNamespaces); got != tt.skip {
				t.Errorf("ignoreItem() = %v, want %v", got, tt.skip)
			}
		})
	}
}
