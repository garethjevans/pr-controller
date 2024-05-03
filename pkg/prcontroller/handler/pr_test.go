package handler_test

import (
	"reflect"
	"testing"

	"github.com/garethjevans/pr-controller/pkg/prcontroller/handler"

	"github.com/garethjevans/pr-controller/pkg/defines"
)

func TestToMap(t *testing.T) {
	type args struct {
		in []defines.GroupVersionResourceKind
	}
	tests := []struct {
		name string
		args args
		want map[defines.GroupVersionResourceKind]defines.GroupVersionResourceKind
	}{
		{
			name: "actual",
			args: args{in: []defines.GroupVersionResourceKind{
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
			want: map[defines.GroupVersionResourceKind]defines.GroupVersionResourceKind{
				{Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackages"}: {Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackageprs"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.ToMap(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
