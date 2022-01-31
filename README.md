# study-grpc
```cd svc```
build:
```protoc --go_out=. --go-grpc_out=. svc.proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative```
build gateway:
```protoc --grpc-gateway_out . --grpc-gateway_opt=logtostderr=true --grpc-gateway_opt=paths=source_relative --grpc-gateway_opt=generate_unbound_methods=true svc.proto```
build openapiv2:
```protoc -I . --openapiv2_out openapiv2 --openapiv2_opt logtostderr=true svc.proto```