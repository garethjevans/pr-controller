package handler

import (
	"github.com/garethjevans/pr-controller/pkg/defines"
	"reflect"
	"testing"
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
				defines.GroupVersionResourceKind{Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackages"}: {Group: "dogfooding.tanzu.broadcom.com", Resource: "carvelpackageprs"},
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
