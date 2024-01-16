package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
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

	// use to cache the port which proxy forward to local
	// the key is a string which combine with hostAddr and proxyAddr
	// e.g: ${hostAddr}_${proxyAddr}
	serviceLocalPortCache     map[string]*portCache
	serviceLocalPortCacheFile *os.File
	// cache each hostAddr tunnel
	// the key is hostAddr
	tunnels map[string]*sshtunnel.SshTunnel
	// use to cache the connected node in each host
	// the key is hostAddr
	connectedMaps map[string][]*connectedNode
}

type connectedNode struct {
	Node   *sshproxypb.Node
	Cancel context.CancelFunc
}

func randomLocalAddr() string {
	l, err := net.Listen("tcp", "")
	if err != nil {
		panic(err)
	}

	defer l.Close()
	return l.Addr().String()
}

func NewServiceTunnel() *ServiceTunnel {
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
		serviceLocalPortCache:     cache,
		serviceLocalPortCacheFile: file,
		tunnels:                   make(map[string]*sshtunnel.SshTunnel),
		connectedMaps:             make(map[string][]*connectedNode),
	}
}

func (st *ServiceTunnel) DialTunnel(tunnel *sshtunnel.SshTunnel) error {
	_, exists := st.tunnels[tunnel.GetRemoteHost()]
	if exists {
		return nil
	}
	st.tunnels[tunnel.GetRemoteHost()] = tunnel
	return nil
}

func (st *ServiceTunnel) Close() {
	for _, connectedNodes := range st.connectedMaps {
		for _, node := range connectedNodes {
			node.Cancel()
		}
	}

	for _, tunnel := range st.tunnels {
		tunnel.Close()
	}

	st.serviceLocalPortCacheFile.Close()
	lg.Info("ServiceTunnel closed")
}

func (st *ServiceTunnel) GetSpecifyRemoteTunnel(host string) (*sshtunnel.SshTunnel, error) {
	tunnel, exists := st.tunnels[host]
	if !exists {
		return nil, fmt.Errorf("host: %v tunnel not exists", host)
	}
	return tunnel, nil
}

func (st *ServiceTunnel) Connect(ctx context.Context, in *sshproxypb.ConnectRequest) (*sshproxypb.ConnectResponse, error) {
	services := in.GetServices()

	connectMaps := st.dialService(ctx, services)

	var nodes []*sshproxypb.Node
	for host, connectMaps := range connectMaps {
		st.connectedMaps[host] = append(st.connectedMaps[host], connectMaps...)
		for _, cn := range connectMaps {
			nodes = append(nodes, cn.Node)
		}
	}

	return &sshproxypb.ConnectResponse{
		ConnectedNodes: nodes,
	}, nil
}

func (st *ServiceTunnel) buildTunnel(ctx context.Context, remoteAddr, proxyAddr, localAddr string) error {
	tunnel, err := st.GetSpecifyRemoteTunnel(remoteAddr)
	if err != nil {
		return errors.Wrap(err, "GetSpecifyRemoteTunnel")
	}

	if err := tunnel.Forward(ctx, localAddr, proxyAddr); err != nil {
		lg.Errorc(ctx, "build tunnel remote: %v -> local: %v error: %v", proxyAddr, localAddr, err)
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

func (st *ServiceTunnel) dialService(ctx context.Context, services []*sshproxypb.Service) map[string][]*connectedNode {
	mappings := make(map[string][]*connectedNode)

	for _, service := range services {
		var localAddr string
		if localPort := st.getLocalPortCache(service.GetProxyAddress()); localPort != "" {
			localAddr = fmt.Sprintf("[::]:%v", localPort)
		} else {
			localAddr = randomLocalAddr()
			_, port, err := net.SplitHostPort(localAddr)
			if err != nil {
				lg.Errorc(ctx, "split local addr error: %v", err)
				continue
			}
			if err := st.writeNewLocalPort(service.GetProxyAddress(), port); err != nil {
				lg.Errorc(ctx, "write local port cache error: %v", err)
				continue
			}
		}
		hostAddr := service.GetRemoteAddress()
		proxyAddr := service.GetProxyAddress()
		ctx, cancel := context.WithCancel(context.TODO())

		lg.Infof("build Tunnel: %v-%v-%v", hostAddr, proxyAddr, localAddr)
		if err := st.buildTunnel(ctx, hostAddr, proxyAddr, localAddr); err != nil {
			lg.Errorf("build tunnel of %v-%v-%v error: %v", hostAddr, proxyAddr, localAddr, err)
			continue
		}

		if _, exists := mappings[service.GetRemoteAddress()]; !exists {
			mappings[service.GetRemoteAddress()] = make([]*connectedNode, 0)
		}

		mappings[service.GetRemoteAddress()] = append(mappings[service.GetRemoteAddress()], &connectedNode{
			Node: &sshproxypb.Node{
				LocalAddress:  localAddr,
				RemoteAddress: proxyAddr,
				HostAddress:   service.GetRemoteAddress(),
				ServiceName:   service.GetServiceName(),
			},
			Cancel: cancel,
		})
	}
	return mappings
}

func (st *ServiceTunnel) Disconnect(ctx context.Context, in *sshproxypb.DisconnectRequest) (*sshproxypb.DisconnectResponse, error) {
	srvs, exists := st.connectedMaps[in.GetHostAddress()]
	if !exists {
		lg.Errorc(ctx, "disconnect host: %v not found", in.GetHostAddress())
		return nil, errors.New("hostAddr has not been dial")
	}

	delIdx := -1
	for idx, srv := range srvs {
		if srv.Node.GetRemoteAddress() == in.GetProxyAddress() {
			srv.Cancel()
			delIdx = idx
			break
		}
	}

	if delIdx != -1 {
		srvs = append(srvs[:delIdx], srvs[delIdx+1:]...)
		st.connectedMaps[in.GetHostAddress()] = srvs
	}

	lg.Infoc(ctx, "disconnect service: %v-%v success", in.GetHostAddress(), in.GetProxyAddress())

	return &sshproxypb.DisconnectResponse{}, nil
}

func (st *ServiceTunnel) GetConnectNodes(ctx context.Context, in *sshproxypb.GetConnectNodesRequest) (*sshproxypb.GetConnectNodesResponse, error) {

	var nodes []*sshproxypb.Node
	for _, connectedNodes := range st.connectedMaps {
		for _, n := range connectedNodes {
			nodes = append(nodes, n.Node)
		}
	}

	return &sshproxypb.GetConnectNodesResponse{
		ConnectedNodes: nodes,
	}, nil
}
