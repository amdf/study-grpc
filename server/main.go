package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/amdf/study-grpc/svc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SimpleServiceServer struct {
	pb.UnimplementedSimpleServiceServer
}

func (server *SimpleServiceServer) SimpleFunction(ctx context.Context, q *pb.SimpleQuery) (result *pb.SimpleResponse, err error) {
	header := metadata.Pairs("header-key", "42")
	if nil != grpc.SendHeader(ctx, header) {
		err = status.Errorf(codes.Unknown, "unable to set header")
	}
	trailer := metadata.Pairs("trailer-key", "3.14")
	if nil != grpc.SetTrailer(ctx, trailer) {
		err = status.Errorf(codes.Unknown, "unable to set header")
	}

	count := len(q.Text)
	result = &pb.SimpleResponse{RuneCount: int32(count)}

	fmt.Println("SimpleFunction. Text = ", q.Text)

	return
}

func (server *SimpleServiceServer) GenerateWords(w *pb.WantWords, stream pb.SimpleService_GenerateWordsServer) error {

	for i := int32(0); i < w.Count; i++ {
		text := "word"
		err := stream.Send(&pb.Word{Text: text, T: timestamppb.New(time.Now())})
		if err != nil {
			return errors.New("error writing to stream")
		}
		time.Sleep(w.Delay.AsDuration())
	}

	return nil
}

func (server *SimpleServiceServer) Sum(stream pb.SimpleService_SumServer) error {
	var result uint64
	for {
		num, err := stream.Recv()
		if err == io.EOF {
			result := pb.SumResult{Value: result}
			stream.SendAndClose(&result)
			break
		}
		if err != nil {
			return errors.New("error reading arguments stream")
		}
		result += uint64(num.Value)
	}
	return nil
}

func (server *SimpleServiceServer) Exchange(stream pb.SimpleService_ExchangeServer) error {
	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		var stop bool
		ticker := time.NewTicker(time.Millisecond * 100 * time.Duration(rand.Intn(8)))
		defer ticker.Stop()

		for !stop {
			select {
			case <-ctx.Done():
				stop = true
			case <-ticker.C:
				t := timestamppb.New(time.Now())
				text := "from server"
				if nil != stream.Send(&pb.SomeText{T: t, Text: text}) {
					stop = true
				}
				fmt.Println(">>>", t.AsTime().Local().Format("15:04:05"), text)
				ticker.Reset(time.Millisecond * 100 * time.Duration(rand.Intn(8)))
			}
		}

		log.Println("Exchange() stop sending")
	}(ctx)

	ipaddr := getAddr(stream.Context())

	for {
		in, err := stream.Recv()
		//if err == io.EOF {
		if err != nil {
			cancel()
			fmt.Println("Exchange() stop receiving")
			break
		}

		fmt.Println("<<<", ipaddr, in.T.AsTime().Local().Format("15:04:05"), in.Text)
	}
	return nil
}

func getAddr(ctx context.Context) string {
	ipaddr := "(unknown)"
	p, ok := peer.FromContext(ctx)
	if ok {
		ipaddr = p.Addr.String()
	}
	return ipaddr
}

func authInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		tokens := md.Get("access-token")
		if len(tokens) > 0 {
			ok = ("token1234" == tokens[0])
		}
		fmt.Println(`-- `, md)
	}

	if ok {
		result, err := handler(ctx, req)
		ipaddr := getAddr(ctx)

		rid := md.Get("Request-ID")
		id := "(nothing)"
		if len(rid) > 0 {
			id = rid[0]
		}

		fmt.Println(ok, "ADDR=", ipaddr, "ID=", id, info.FullMethod)
		return result, err
	}

	return nil, status.Errorf(codes.PermissionDenied, "Auth Error")
}

func main() {
	var server SimpleServiceServer
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalln("fail to listen", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
	)
	pb.RegisterSimpleServiceServer(s, &server)

	log.Println("starting server at ", lis.Addr())
	err = s.Serve(lis)
	if err != nil {
		log.Fatalln("fail to server", err)
	}
}
