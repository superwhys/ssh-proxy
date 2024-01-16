# ssh-proxy 

A ssh proxy utils developed by GO 

In the current version, it only provides forward proxy functionality

## Install

```bash
go install github.com/superwhys/ssh-proxy
```

### protobuf generate
#### sshproxy.proto
```bash
protoc sshproxypb/sshproxy.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=.
```

## Usage
It provide two ways to proxy remote port 

### Direct connect

**single**
```bash
ssh-proxy connect sshHost:sshPort remoteHost:remotePort
```

You can proxy the specified port of the remote sshHost locally, as in the example above.

**multi**
```bash
ssh-proxy connect sshHost1:sshPort1 remoteHost:remotePort sshHost2:sshPort2 remoteHost:remotePort
```

You can also proxy multiple different remote ports locally at the same time.

### Mesh connect

```bash
Available Commands:
  append      Append services to existing mesh
  connect     Build tunnel to set of services
  create      Create a mesh of multiple services
  delete      Delete mesh
  ls          List all mesh services
```

It provides command like these

Before use this mode, you need to create a mesh.

Before create a mesh, you should has a env config `(.ssh-proxy.yaml)` like this in you `HOME` dir
```text
profiles:
  - EnvName: dev
    # you can just one host in this
    # if you provide muti host, it will use the previous host as a jumpers.
    Hosts:
      - HostName: xxx.xxx.xxx.xxx 
      - HostName: yyy.yyy.yyy.yyy
        IdentityFile: ~/.ssh/id_rsa
```

then, you can create a mesh very simply

```bash
ssh-proxy mesh create --env dev mesh-test localhost:8000
```

and you can connect to mesh like that

```bash
ssh-proxy mesh connect mesh-test
```

### GRPC-UI

after you proxy the remote port locally, it will start a grpc server and provide a grpcui debug page,

in this page, there are there command:  `connect`, `disconnect`, `getAllNodes` for you to monitor your proxy
