package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/ssh-proxy/sshproxypb"
	"github.com/superwhys/sshtunnel"
)

var (
	localPortCacheFile = "/tmp/.service-tunnel-local-port-cache"
)

type portCache struct {
	RemoteAddr string
	LocalPort  string
}

type ServiceTunnel struct {
	sshproxypb.UnimplementedServiceTunnelServer

	serviceLocalPortCache     map[string]*portCache
	serviceLocalPortCacheFile *os.File
	tunnel                    *sshtunnel.SshTunnel
	connectedMaps             map[string]*connectedNodes
}

type connectedNodes struct {
	ServiceName string
	Nodes       []*sshproxypb.Node
	Cancel      context.CancelFunc
}

func randomLocalAddr() string {
	l, _ := net.Listen("tcp", "")
	defer l.Close()
	return l.Addr().String()
}

func NewServiceTunnel(tunnel *sshtunnel.SshTunnel) *ServiceTunnel {
	file, err := os.OpenFile(localPortCacheFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	lg.PanicError(err)
	data, err := io.ReadAll(file)
	lg.PanicError(err)

	cache := make(map[string]*portCache)
	for _, line := range strings.Split(string(data), "\n") {
		lineSplit := strings.Split(line, "-")
		if len(lineSplit) != 2 {
			continue
		}

		cache[lineSplit[0]] = &portCache{
			RemoteAddr: lineSplit[0],
			LocalPort:  lineSplit[1],
		}
	}

	return &ServiceTunnel{
		tunnel:                    tunnel,
		serviceLocalPortCache:     cache,
		serviceLocalPortCacheFile: file,
		connectedMaps:             make(map[string]*connectedNodes),
	}
}

func (st *ServiceTunnel) Close() {
	for _, connectedNodes := range st.connectedMaps {
		connectedNodes.Cancel()
	}
	st.tunnel.Close()
	st.serviceLocalPortCacheFile.Close()
	lg.Info("ServiceTunnel closed")
}

func (st *ServiceTunnel) GetRemoteHost() string {
	return st.tunnel.GetRemoteHost()
}

func (st *ServiceTunnel) Connect(ctx context.Context, in *sshproxypb.ConnectRequest) (*sshproxypb.ConnectResponse, error) {
	services := in.GetServices()
	serviceName := in.GetServiceName()

	ctx, cancel := context.WithCancel(ctx)
	connectMaps := st.dialService(ctx, services, serviceName)

	st.connectedMaps[st.tunnel.GetRemoteHost()] = &connectedNodes{
		ServiceName: serviceName,
		Nodes:       connectMaps,
		Cancel:      cancel,
	}

	return &sshproxypb.ConnectResponse{
		ConnectedNodes: connectMaps,
	}, nil
}

func (st *ServiceTunnel) buildTunnel(ctx context.Context, remoteAddr string, localAddr string) error {
	if err := st.tunnel.Forward(ctx, localAddr, remoteAddr); err != nil {
		lg.Errorc(ctx, "build tunnel remote: %v -> local: %v error: %v", remoteAddr, localAddr, err)
		return err
	}

	return nil
}

func (st *ServiceTunnel) getLocalPortCache(remoteAddr string) string {
	if cache, ok := st.serviceLocalPortCache[remoteAddr]; ok {
		return cache.LocalPort
	}

	return ""
}

func (st *ServiceTunnel) writeNewLocalPort(remoteAddr string, localPort string) error {
	if _, ok := st.serviceLocalPortCache[remoteAddr]; ok {
		return nil
	}

	st.serviceLocalPortCache[remoteAddr] = &portCache{
		RemoteAddr: remoteAddr,
		LocalPort:  localPort,
	}
	st.serviceLocalPortCacheFile.WriteString(fmt.Sprintf("%v-%v\n", remoteAddr, localPort))
	return nil
}

func (st *ServiceTunnel) dialService(ctx context.Context, services []*sshproxypb.Service, serviceName string) []*sshproxypb.Node {
	var mappings []*sshproxypb.Node

	for _, service := range services {
		var localAddr string
		if localPort := st.getLocalPortCache(service.GetRemoteAddress()); localPort != "" {
			localAddr = fmt.Sprintf("[::]:%v", localPort)
		} else {
			localAddr = randomLocalAddr()
			_, port, err := net.SplitHostPort(localAddr)
			if err != nil {
				lg.Errorc(ctx, "split local addr error: %v", err)
				continue
			}
			if err := st.writeNewLocalPort(service.GetRemoteAddress(), port); err != nil {
				lg.Errorc(ctx, "write local port cache error: %v", err)
				continue
			}
		}

		remoteAddr := service.GetRemoteAddress()
		if err := st.buildTunnel(ctx, remoteAddr, localAddr); err != nil {
			continue
		}

		mappings = append(mappings, &sshproxypb.Node{
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,
			ServiceName:   service.GetServiceName(),
		})
	}
	return mappings
}

func (st *ServiceTunnel) Disconnect(ctx context.Context, in *sshproxypb.DisconnectRequest) (*sshproxypb.DisconnectResponse, error) {
	serviceName := in.GetServiceName()
	if _, ok := st.connectedMaps[serviceName]; !ok {
		lg.Errorc(ctx, "disconnect service: %v not found", serviceName)
		return nil, errors.New("service not found")
	}

	st.connectedMaps[serviceName].Cancel()
	delete(st.connectedMaps, serviceName)
	lg.Infoc(ctx, "disconnect service: %v success", serviceName)

	return &sshproxypb.DisconnectResponse{}, nil
}

func (st *ServiceTunnel) GetConnectNodes(ctx context.Context, in *sshproxypb.GetConnectNodesRequest) (*sshproxypb.GetConnectNodesResponse, error) {

	var nodes []*sshproxypb.Node
	for _, connectedNodes := range st.connectedMaps {
		nodes = append(nodes, connectedNodes.Nodes...)
	}

	return &sshproxypb.GetConnectNodesResponse{
		ConnectedNodes: nodes,
	}, nil
}
