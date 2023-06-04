package main

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseLabelsFlag(t *testing.T) {
	tests := []struct {
		name       string
		labelsFlag string
		want       map[string]string
	}{
		{
			name: "empty",
		},
		{
			name:       "happy",
			labelsFlag: "key0=value0,key1=value1",
			want:       map[string]string{"key0": "value0", "key1": "value1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLabelsFlag(tt.labelsFlag); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLabelsFlag() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSkipLabels(t *testing.T) {
	type args struct {
		gotLabels    map[string]string
		wantLabels   map[string]string
		ignoreLabels map[string]string
	}
	tests := []struct {
		name string
		args args
		skip bool
	}{
		{
			name: "empty",
			skip: false,
		},
		{
			name: "no want/ignore",
			args: args{gotLabels: map[string]string{"key0": "value0"}},
			skip: false,
		},
		{
			name: "want same",
			args: args{
				gotLabels:  map[string]string{"key0": "value0"},
				wantLabels: map[string]string{"key0": "value0"},
			},
			skip: false,
		},
		{
			name: "want different",
			args: args{
				gotLabels:  map[string]string{"key0": "value0"},
				wantLabels: map[string]string{"key3": "value3"},
			},
			skip: true,
		},
		{
			name: "ignore same",
			args: args{
				gotLabels:    map[string]string{"key0": "value0"},
				ignoreLabels: map[string]string{"key0": "value0"},
			},
			skip: true,
		},
		{
			name: "ignore different",
			args: args{
				gotLabels:    map[string]string{"key0": "value0"},
				ignoreLabels: map[string]string{"key3": "value3"},
			},
			skip: false,
		},
		{
			name: "multiple want",
			args: args{
				gotLabels:    map[string]string{"key0": "value0", "key1": "value1", "key2": "value2"},
				wantLabels:   map[string]string{"key3": "value3", "key1": "value1"},
				ignoreLabels: map[string]string{"key3": "value3"},
			},
			skip: false,
		},
		{
			name: "multiple ignore",
			args: args{
				gotLabels:    map[string]string{"key0": "value0", "key1": "value1", "key2": "value2"},
				wantLabels:   map[string]string{"key3": "value3", "key1": "value1"},
				ignoreLabels: map[string]string{"key1": "value1", "key3": "value3"},
			},
			skip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipLabels(tt.args.gotLabels, tt.args.wantLabels, tt.args.ignoreLabels); got != tt.skip {
				t.Errorf("skipLabels() = %v, want %v", got, tt.skip)
			}
		})
	}
}

func TestSkipGroup(t *testing.T) {
	type args struct {
		group        metav1.APIGroup
		wantGroups   []string
		ignoreGroups []string
	}
	tests := []struct {
		name string
		args args
		skip bool
	}{
		{
			name: "empty",
			skip: false,
		},
		{
			name: "no want/ignore",
			args: args{group: metav1.APIGroup{Name: "currentGroup"}},
			skip: false,
		},
		{
			name: "not wanted",
			args: args{
				group:      metav1.APIGroup{Name: "currentGroup"},
				wantGroups: []string{"notCurrentGroup"},
			},
			skip: true,
		},
		{
			name: "wanted",
			args: args{
				group:      metav1.APIGroup{Name: "currentGroup"},
				wantGroups: []string{"currentGroup"},
			},
			skip: false,
		},
		{
			name: "ignored",
			args: args{
				group:        metav1.APIGroup{Name: "currentGroup"},
				ignoreGroups: []string{"currentGroup"},
			},
			skip: true,
		},
		{
			name: "not ignored",
			args: args{
				group:        metav1.APIGroup{Name: "currentGroup"},
				ignoreGroups: []string{"notCurrentGroup"},
			},
			skip: false,
		},
		{
			name: "wanted and ignored",
			args: args{
				group:        metav1.APIGroup{Name: "currentGroup"},
				wantGroups:   []string{"currentGroup"},
				ignoreGroups: []string{"currentGroup"},
			},
			skip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipGroup(tt.args.group, tt.args.wantGroups, tt.args.ignoreGroups); got != tt.skip {
				t.Errorf("skipGroup() = %v, want %v", got, tt.skip)
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
			name: "verb list",
			args: args{
				res: metav1.APIResource{Verbs: metav1.Verbs{"list"}},
			},
			skip: false,
		},
		{
			name: "subresource",
			args: args{
				res: metav1.APIResource{Name: "resource/subresource", Verbs: metav1.Verbs{"list"}},
			},
			skip: true,
		},

		{
			name: "empty string want/ignore",
			args: args{
				res: metav1.APIResource{
					Name:  "myresource",
					Verbs: metav1.Verbs{"list"},
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
					Verbs: metav1.Verbs{"list"},
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
					Verbs: metav1.Verbs{"list"},
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
					Verbs: metav1.Verbs{"list"},
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
					Verbs: metav1.Verbs{"list"},
				},
				ignoreResources: []string{"myresource"},
			},
			skip: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipResource(tt.args.res, tt.args.wantResources, tt.args.ignoreResources); got != tt.skip {
				t.Errorf("skipResource() = %v, want %v", got, tt.skip)
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
				t.Errorf("skipItem() = %v, want %v", got, tt.skip)
			}
		})
	}
}
