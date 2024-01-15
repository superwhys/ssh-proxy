package server

import (
	"os"
	"reflect"
	"testing"

	"github.com/superwhys/goutils/lg"
)

func TestMain(m *testing.M) {
	lg.PanicError(os.Remove(meshFile))
	m.Run()
}

func TestServiceMesh_CreateMesh(t *testing.T) {
	type args struct {
		mesh Mesh
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "CreateMesh-1", args: args{mesh: Mesh{Name: "mesh-1", Env: "env-1", Services: []Service{{ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}}}},
		{name: "CreateMesh-2", args: args{mesh: Mesh{Name: "mesh-2", Env: "env-2", Services: []Service{{ServiceName: "service-2", RemoteAddr: "remote-addr-2"}}}}},
		{name: "CreateMesh-3", args: args{mesh: Mesh{Name: "mesh-3", Env: "env-1", Services: []Service{{ServiceName: "service-2", RemoteAddr: "remote-addr-2"}, {ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMesh{}
			s.CreateMesh(tt.args.mesh)
		})
	}
}

func TestServiceMesh_GetAllMeshes(t *testing.T) {
	tests := []struct {
		name string
		want []Mesh
	}{
		{
			name: "GetAllMeshes-1", want: []Mesh{
				{Name: "mesh-1", Env: "env-1", Services: []Service{{ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}},
				{Name: "mesh-2", Env: "env-2", Services: []Service{{ServiceName: "service-2", RemoteAddr: "remote-addr-2"}}},
				{Name: "mesh-3", Env: "env-1", Services: []Service{{ServiceName: "service-2", RemoteAddr: "remote-addr-2"}, {ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMesh{}

			got, err := s.GetAllMeshes()
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceMesh.GetAllMeshes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMesh_GetMesh(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *Mesh
	}{
		{name: "GetMesh-1", args: args{name: "mesh-1"}, want: &Mesh{Name: "mesh-1", Env: "env-1", Services: []Service{{ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}}},
		{name: "GetMesh-2", args: args{name: "mesh-3"}, want: &Mesh{Name: "mesh-3", Env: "env-1", Services: []Service{{ServiceName: "service-2", RemoteAddr: "remote-addr-2"}, {ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMesh{}
			got, err := s.GetMesh(tt.args.name)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceMesh.GetMesh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMesh_AddServiceToMesh(t *testing.T) {
	type args struct {
		meshName string
		service  *Service
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "AddServiceToMesh-1", args: args{meshName: "mesh-1", service: &Service{ServiceName: "service-1", RemoteAddr: "remote-addr-1"}}},
		{name: "AddServiceToMesh-2", args: args{meshName: "mesh-1", service: &Service{ServiceName: "service-1-2", RemoteAddr: "remote-addr-1-2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMesh{}
			s.AddServiceToMesh(tt.args.meshName, tt.args.service)
		})
	}
}

func TestServiceMesh_DeleteMesh(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		s    *ServiceMesh
		args args
	}{
		{name: "DeleteMesh-1", s: &ServiceMesh{}, args: args{name: "mesh-2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMesh{}
			s.DeleteMesh(tt.args.name)
		})
	}
}
