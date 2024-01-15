# service-tunnel

## protobuf generate
#### sshproxy.proto
```shell
protoc sshproxypb/sshproxy.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=.
```
