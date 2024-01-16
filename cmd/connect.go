/*
Copyright © 2023 Yong
*/
package cmd

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
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
	Use:   "connect [options] [sshHost:sshPort proxyHost:proxyPort...] | [proxyHost:proxyPort]",
	Short: "Proxy the proxyHost:proxyPort to the local through sshHost",
	Long: `Proxy the proxyHost:proxyPort to the local through sshHost. 
	It is similar to a forward proxy for ssh.
	You can provide the remote side which need to be connect and the proxy side to proxy the service to local like:

	ssh-proxy sshHost:sshPort localhost:8000

	or 
	You can use the profile config to define some alias of remote side, so that use can just connect to remote like:

	ssh-proxy --env aliasName localhost:8000

	`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		user := flags.String("user", "root", "")

		flags.Parse()

		var err error
		if env() == "" {
			if len(args)%2 != 0 {
				return errors.New("args is not a valid sshHost and proxyHost pairs")
			}

			proxyHosts, err := parseHostPortPairs(args...)
			if err != nil {
				return errors.Wrap(err, "parse host pairs")
			}

			err = startConnectDirect(user(), privateKeyPath(), proxyHosts)
		} else {
			proxyHosts, err := parseProfileHostPort(args...)
			if err != nil {
				return errors.Wrap(err, "parse profile hostPort")
			}
			err = startConnect(proxyHosts)
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

	lg.Info(lg.Jsonify(profile))

	return sshtunnel.NewTunnel(profile.Hosts...)
}

func startConnectDirect(user, identityFile string, proxyHosts []*sshproxypb.Service) error {
	lg.Info("connect direct")

	table := map[string][]*sshproxypb.Node{}
	serviceTunnel := server.NewServiceTunnel()
	connectServices := make([]*sshproxypb.Service, 0)
	tunnelCache := make(map[string]*sshtunnel.SshTunnel)
	defer func() {
		for _, st := range tunnelCache {
			st.Close()
		}
		serviceTunnel.Close()
	}()

	ctx := context.Background()

	for _, pair := range proxyHosts {
		sshHost := pair.RemoteAddress

		if _, exists := tunnelCache[sshHost]; !exists {
			tunnel := dialDirectTunnel(user, sshHost, identityFile)
			lg.Infof("dial ssh tunnel success: %v", sshHost)
			serviceTunnel.DialTunnel(tunnel)
			tunnelCache[sshHost] = tunnel
		}

		connectServices = append(connectServices, pair)
	}

	resp, err := serviceTunnel.Connect(ctx, &sshproxypb.ConnectRequest{
		Services: connectServices,
	})
	if err != nil {
		lg.Errorc(ctx, "Failed to connect remote services: %v", err)
		return errors.Wrap(err, "tunnelConnect")
	}
	for _, node := range resp.GetConnectedNodes() {
		table[node.GetHostAddress()] = append(table[node.GetHostAddress()], node)
	}

	lg.Info("Connected services\n" + prettyMaps(table))

	serviceOpts := []service.SuperServiceOption{
		service.WithGRPCUI(),
		service.WithPprof(),
	}

	serviceOpts = append(serviceOpts, service.WithGRPC(func(srv *grpc.Server) {
		sshproxypb.RegisterServiceTunnelServer(srv, serviceTunnel)
	}))

	srv := service.NewSuperService(serviceOpts...)
	return srv.ListenAndServer(port())
}

// startConnect used to connect remote services with tunnel
// By default, all services connected at a single time are under the same host
func startConnect(proxyHosts []*sshproxypb.Service) error {
	ctx := context.Background()
	tunnel, err := dialTunnel()
	if err != nil {
		return err
	}

	for _, ph := range proxyHosts {
		ph.RemoteAddress = tunnel.GetRemoteHost()
	}

	st := server.NewServiceTunnel()
	st.DialTunnel(tunnel)
	defer st.Close()
	resp, err := st.Connect(ctx, &sshproxypb.ConnectRequest{
		Services: proxyHosts,
	})
	if err != nil {
		lg.Errorc(ctx, "Failed to connect remote services: %v", err)
		return err
	}

	table := map[string][]*sshproxypb.Node{}

	for _, node := range resp.GetConnectedNodes() {
		table[tunnel.GetRemoteHost()] = append(table[tunnel.GetRemoteHost()], node)
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

	connectCmd.Flags().StringP("user", "u", "root", "User to connect to remote services.")
}
