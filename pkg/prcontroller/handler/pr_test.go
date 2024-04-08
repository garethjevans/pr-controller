package handler

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"testing"
)

func TestToMap(t *testing.T) {
	type args struct {
		in []schema.GroupResource
	}
	tests := []struct {
		name string
		args args
		want map[schema.GroupResource]*schema.GroupResource
	}{
		{
			name: "actual",
			args: args{in: []schema.GroupResource{
				{
					Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackageprs",
				},
				{
					Group: "dogfooding.tanzu.broadcom.com", Resource: "renovates",
				},
				{
					Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackages",
				},
				{
					Group: "supplychain.app.tanzu.vmware.com", Resource: "containerappworkflows",
				},
			}},
			want: map[schema.GroupResource]*schema.GroupResource{
				schema.GroupResource{Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackages"}: {Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackageprs"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToMap(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
