all: svc.swagger.json

svc.swagger.json: svc.pb.gw.go
	protoc -I . --openapiv2_out . --openapiv2_opt logtostderr=true svc.proto

svc.pb.gw.go: svc_grpc.pb.go

svc_grpc.pb.go: svc.pb.go
	protoc --grpc-gateway_out . --grpc-gateway_opt=logtostderr=true --grpc-gateway_opt=paths=source_relative --grpc-gateway_opt=generate_unbound_methods=true svc.proto

svc.pb.go: svc.proto
	protoc --go_out=. --go-grpc_out=. svc.proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative

clean: end
	$(RM) svc.pb.go svc_grpc.pb.go svc.pb.gw.go
	$(RM) svc.swagger.json

end: