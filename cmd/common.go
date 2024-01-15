package cmd

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/superwhys/ssh-proxy/sshproxypb"
)

func prettySlices(slice []string) string {
	buffer := &bytes.Buffer{}
	table := tablewriter.NewWriter(buffer)

	var current []string
	for i, s := range slice {
		current = append(current, s)
		if (i+1)%5 == 0 {
			table.Append(current)
			current = nil
		}
	}
	if len(current) > 0 {
		table.Append(current)
	}
	table.Render()
	return buffer.String()
}

func prettyMaps(m map[string][]*sshproxypb.Node) string {
	buffer := &bytes.Buffer{}
	table := tablewriter.NewWriter(buffer)

	type Record struct {
		Host          string
		ServiceName   string
		RemoteAddress string
		Port          string
		DebugURL      string
	}
	var rs []*Record
	for host, connectNode := range m {
		for _, node := range connectNode {
			_, port, _ := net.SplitHostPort(node.GetLocalAddress())
			r := &Record{
				Host:          host,
				ServiceName:   node.GetServiceName(),
				RemoteAddress: node.GetRemoteAddress(),
				Port:          port,
				DebugURL:      prettyLocalAddr(node.GetLocalAddress()),
			}
			rs = append(rs, r)
		}
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Host == rs[j].Host {
			if rs[i].ServiceName == rs[j].ServiceName {
				return rs[i].Port < rs[j].Port
			}
			return rs[i].ServiceName < rs[j].ServiceName
		}

		return rs[i].Host < rs[j].Host
	})
	table.Append([]string{"Host", "Service", "Remote Address", "Local Port", "Debug URL"})
	for _, r := range rs {
		table.Append([]string{r.Host, r.ServiceName, r.RemoteAddress, r.Port, r.DebugURL})
	}
	table.Render()
	return buffer.String()
}

func prettyLocalAddr(addr string) string {
	addr = strings.Replace(addr, "[::]:", "", -1)
	addr = strings.Replace(addr, "127.0.0.1:", "", -1)
	return "http://localhost:" + addr + "/debug"
}

func parseHostPortPairs(args ...string) ([]*sshproxypb.Service, error) {
	var formatServices []*sshproxypb.Service
	dupService := make(map[string]bool)

	appendService := func(serviceName, remoteAddr string) {
		if _, ok := dupService[serviceName]; ok {
			return
		}
		formatServices = append(formatServices, &sshproxypb.Service{
			ServiceName:   serviceName,
			RemoteAddress: remoteAddr,
		})
		dupService[serviceName] = true
	}

	for i := 0; i < len(args); {
		// first arg in a group is service name
		if !isTCPAddr(args[i]) {
			if isTCPAddr(args[i+1]) {
				appendService(args[i], args[i+1])
				i++
			} else {
				return nil, fmt.Errorf("A service name must be followed by a remote address: %s", args[i])
			}
		} else {
			// first arg in a group is remote address
			appendService(args[i], args[i])
		}
		i++
	}

	// for _, arg := range args {
	// 	_, _, err := net.SplitHostPort(arg)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	appendService(arg, arg)
	// }

	return formatServices, nil
}
