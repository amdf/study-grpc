package main

import (
	"context"
	"log"
	"net"

	pb "github.com/amdf/study-grpc/svc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SimpleServiceServer struct {
	pb.UnimplementedSimpleServiceServer
}

func (server *SimpleServiceServer) SimpleFunction(ctx context.Context, q *pb.SimpleQuery) (result *pb.SimpleResponse, err error) {
	return nil, status.Errorf(codes.Unimplemented, "method SimpleFunction not implemented")
}
func (server *SimpleServiceServer) GenerateWords(w *pb.WantWords, gwServer pb.SimpleService_GenerateWordsServer) error {
	return status.Errorf(codes.Unimplemented, "method GenerateWords not implemented")
}
func (server *SimpleServiceServer) Sum(sumServer pb.SimpleService_SumServer) error {
	return status.Errorf(codes.Unimplemented, "method Sum not implemented")
}
func (server *SimpleServiceServer) Exchange(exServer pb.SimpleService_ExchangeServer) error {
	return status.Errorf(codes.Unimplemented, "method Exchange not implemented")
}

func main() {
	var server SimpleServiceServer
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalln("fail to listen", err)
	}
	s := grpc.NewServer()
	pb.RegisterSimpleServiceServer(s, &server)
	log.Println("starting server at ", lis.Addr())
	err = s.Serve(lis)
	if err != nil {
		log.Fatalln("fail to server", err)
	}
}
