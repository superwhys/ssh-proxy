/*
Copyright © 2023 Yong
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.com/superwhys/ssh-proxy/server"
	"github.com/superwhys/ssh-proxy/sshproxypb"

	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/goutils/service"
	"github.com/superwhys/sshtunnel"
	"google.golang.org/grpc"
)

func isTCPAddr(addr string) bool {
	_, _, err := net.SplitHostPort(addr)
	return err == nil
}

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect <host:port | alias host:port>..",
	Short: "Connects specified port in host",
	RunE: func(cmd *cobra.Command, args []string) error {
		useProfile := flags.Bool("useProfile", false, "whether to connect with specified profile")
		tunnelPort := flags.Int("tunnelPort", 22, "ssh tunnel connect port")
		user := flags.String("user", "root", "")

		flags.Parse()

		lg.Infof("useProfile: %v, tunnelPort: %v", useProfile(), tunnelPort())

		formatServices, err := parseHostPortPairs(args...)
		if err != nil {
			return err
		}

		if len(formatServices) == 0 {
			return errors.New("connect command need service provided")
		}

		if !useProfile() {
			err = startConnectDirect(tunnelPort(), user(), privateKeyPath(), formatServices)
		} else {
			err = startConnect(formatServices)
		}
		if err != nil {
			lg.Errorf("Failed to start connect: %v", err)
			os.Exit(1)
		}
		return nil
	},
}

func dialTunnel() (*sshtunnel.SshTunnel, error) {
	var profile *ConnectionProfile
	var allProfiles []*ConnectionProfile
	profiles(&allProfiles)
	for _, p := range allProfiles {
		if p.EnvName == env() {
			profile = p
			break
		}
	}
	if profile == nil {
		return nil, fmt.Errorf("No connection profile found. env=%s", env())
	}
	profile.PopulateDefault(privateKeyPath())

	lg.Infof("Connecting remote services with profile:\n%s", lg.Jsonify(profile))
	tunnel := sshtunnel.NewTunnel(profile.Hosts...)

	return tunnel, nil
}

func dialDirectTunnel(user, host, identityFile string) *sshtunnel.SshTunnel {
	profile := &ConnectionProfile{
		EnvName: "direct",
		Hosts: []*sshtunnel.SshConfig{
			{HostName: host, User: user, IdentityFile: identityFile},
		},
	}

	return sshtunnel.NewTunnel(profile.Hosts...)
}

func startConnectDirect(tunnelPort int, user, identityFile string, args []*sshproxypb.Service) error {
	ctx := context.Background()

	table := map[string][]*sshproxypb.Node{}
	serviceTunnelGroup := make(map[string]*server.ServiceTunnel)
	defer func() {
		for _, st := range serviceTunnelGroup {
			st.Close()
		}
	}()

	for _, arg := range args {
		host, _, _ := net.SplitHostPort(arg.RemoteAddress)
		host = fmt.Sprintf("%v:%v", host, tunnelPort)

		var serviceTunnel *server.ServiceTunnel
		if serviceTunnel = serviceTunnelGroup[host]; serviceTunnel == nil {
			tunnel := dialDirectTunnel(user, host, identityFile)
			serviceTunnel = server.NewServiceTunnel(tunnel)
			serviceTunnelGroup[host] = serviceTunnel
		}

		resp, err := serviceTunnel.Connect(ctx, &sshproxypb.ConnectRequest{
			Services: []*sshproxypb.Service{arg},
		})
		if err != nil {
			lg.Errorc(ctx, "Failed to connect remote services: %v", err)
			continue
		}
		for _, node := range resp.GetConnectedNodes() {
			table[serviceTunnel.GetRemoteHost()] = append(table[serviceTunnel.GetRemoteHost()], node)
		}
	}

	lg.Info("Connected services\n" + prettyMaps(table))

	serviceOpts := []service.SuperServiceOption{
		service.WithGRPCUI(),
		service.WithPprof(),
	}
	for _, st := range serviceTunnelGroup {
		serviceOpts = append(serviceOpts, service.WithGRPC(func(srv *grpc.Server) {
			sshproxypb.RegisterServiceTunnelServer(srv, st)
		}))
	}

	srv := service.NewSuperService(serviceOpts...)
	return srv.ListenAndServer(port())
}

// startConnect used to connect remote services with tunnel
// By default, all services connected at a single time are under the same host
func startConnect(args []*sshproxypb.Service) error {
	ctx := context.Background()
	tunnel, err := dialTunnel()
	if err != nil {
		return err
	}

	st := server.NewServiceTunnel(tunnel)
	defer st.Close()
	resp, err := st.Connect(ctx, &sshproxypb.ConnectRequest{
		Services: args,
	})
	if err != nil {
		lg.Errorc(ctx, "Failed to connect remote services: %v", err)
		return err
	}

	// map{"remoteHost": [{local, remote, serviceName, tag}]}
	table := map[string][]*sshproxypb.Node{}

	for _, node := range resp.GetConnectedNodes() {
		table[st.GetRemoteHost()] = append(table[st.GetRemoteHost()], node)
	}
	lg.Info("Connected services\n" + prettyMaps(table))

	srv := service.NewSuperService(
		service.WithGRPC(func(srv *grpc.Server) {
			sshproxypb.RegisterServiceTunnelServer(srv, st)
		}),
		service.WithGRPCUI(),
		service.WithPprof(),
	)

	return srv.ListenAndServer(port())
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().Bool("useProfile", false, "whether to connect with specified profile")
	connectCmd.Flags().IntP("tunnelPort", "p", 22, "ssh tunnel connect port")
	connectCmd.Flags().StringP("user", "u", "root", "User to connect to remote services.")
}
