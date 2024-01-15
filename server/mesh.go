package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/superwhys/goutils/lg"
)

var (
	meshFile = fmt.Sprintf("%v/.service-tunnel-mesh.json", os.Getenv("HOME"))
	rwcFlag  = os.O_RDWR | os.O_CREATE
)

type ServiceMesh struct {
}

type Service struct {
	ServiceName string
	RemoteAddr  string
}

type Mesh struct {
	Name     string
	Env      string
	Services []Service
}

func NewServiceMesh() *ServiceMesh {
	return &ServiceMesh{}
}

func (s *ServiceMesh) openMeshFile(flag int) *os.File {
	file, err := os.OpenFile(meshFile, flag, 0666)
	lg.PanicError(err)

	return file
}

func (s *ServiceMesh) writeMeshFile(file *os.File, data []byte) error {
	lg.PanicError(file.Truncate(0))

	_, err := file.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "seek file error")
	}

	_, err = file.Write(data)
	if err != nil {
		return errors.Wrap(err, "write file error")
	}

	return nil
}

func (s *ServiceMesh) getAllMeshes(file *os.File) ([]byte, error) {
	return io.ReadAll(file)
}

func (s *ServiceMesh) parseMeshesByte(b []byte) ([]Mesh, error) {
	if len(b) == 0 {
		return nil, nil
	}

	var meshes []Mesh
	err := json.Unmarshal(b, &meshes)
	if err != nil {
		return nil, errors.Wrap(err, "parse meshes error")
	}

	return meshes, nil
}

func (s *ServiceMesh) GetAllMeshes() ([]Mesh, error) {
	file := s.openMeshFile(rwcFlag)
	defer file.Close()

	b, err := s.getAllMeshes(file)
	if err != nil {
		return nil, errors.Wrap(err, "get all meshes error")
	}
	return s.parseMeshesByte(b)
}

func (s *ServiceMesh) CreateMesh(mesh Mesh) error {
	file := s.openMeshFile(rwcFlag)
	defer file.Close()

	b, err := s.getAllMeshes(file)
	if err != nil {
		return errors.Wrap(err, "get all meshes error")
	}
	meshes, err := s.parseMeshesByte(b)
	if err != nil {
		return errors.Wrap(err, "parse meshes error")
	}

	for _, m := range meshes {
		if m.Name == mesh.Name {
			lg.Infof("mesh: %v already exists", mesh.Name)
			return nil
		}
	}

	meshes = append(meshes, mesh)

	b, err = json.Marshal(meshes)
	if err != nil {
		return errors.Wrap(err, "marshal meshes error")
	}

	return s.writeMeshFile(file, b)
}

func (s *ServiceMesh) GetMesh(name string) (*Mesh, error) {
	file := s.openMeshFile(rwcFlag)
	defer file.Close()

	originBytes, err := s.getAllMeshes(file)
	if err != nil {
		return nil, errors.Wrap(err, "get all meshes error")
	}
	meshes, err := s.parseMeshesByte(originBytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse meshes error")
	}

	for _, m := range meshes {
		if m.Name == name {
			return &m, nil
		}
	}

	return nil, errors.New("mesh not exists")
}

func (s *ServiceMesh) AddServiceToMesh(meshName string, service *Service) error {
	file := s.openMeshFile(rwcFlag)
	defer file.Close()

	b, err := s.getAllMeshes(file)
	if err != nil {
		return errors.Wrap(err, "get all meshes error")
	}
	meshes, err := s.parseMeshesByte(b)
	if err != nil {
		return errors.Wrap(err, "parse meshes error")
	}

	var mesh *Mesh
	for i, m := range meshes {
		if m.Name == meshName {
			mesh = &meshes[i]
			break
		}
	}

	if mesh == nil {
		lg.Infof("mesh: %v not exists", service.ServiceName)
		return errors.New("mesh not exists")
	}

	for _, srv := range mesh.Services {
		if srv.RemoteAddr == service.RemoteAddr {
			lg.Infof("service: %v already exists", service.RemoteAddr)
			return nil
		}
	}

	mesh.Services = append(mesh.Services, *service)
	b, err = json.Marshal(meshes)
	if err != nil {
		return errors.Wrap(err, "marshal meshes error")
	}

	return s.writeMeshFile(file, b)
}

func (s *ServiceMesh) DeleteMesh(name string) error {
	file := s.openMeshFile(rwcFlag)
	defer file.Close()

	b, err := s.getAllMeshes(file)
	if err != nil {
		return errors.Wrap(err, "get all meshes error")
	}
	meshes, err := s.parseMeshesByte(b)
	if err != nil {
		return errors.Wrap(err, "parse meshes error")
	}

	var mesh *Mesh
	for i, m := range meshes {
		if m.Name == name {
			mesh = &meshes[i]
			break
		}
	}

	if mesh == nil {
		lg.Infof("mesh: %v not exists", name)
		return errors.New("mesh not exists")
	}

	for i, m := range meshes {
		if m.Name == name {
			meshes = append(meshes[:i], meshes[i+1:]...)
			break
		}
	}

	b, err = json.Marshal(meshes)
	if err != nil {
		return errors.Wrap(err, "marshal meshes error")
	}

	return s.writeMeshFile(file, b)
}
