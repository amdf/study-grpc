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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SimpleServiceServer struct {
	pb.UnimplementedSimpleServiceServer
}

func (server *SimpleServiceServer) SimpleFunction(ctx context.Context, q *pb.SimpleQuery) (result *pb.SimpleResponse, err error) {

	count := len(q.Text)
	result = &pb.SimpleResponse{RuneCount: int32(count)}
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
				if nil != stream.Send(&pb.SomeText{T: t, Text: "this is text"}) {
					stop = true
				}
				ticker.Reset(time.Millisecond * 100 * time.Duration(rand.Intn(8)))
			}
		}

		log.Println("Exchange() stop sending")
	}(ctx)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			cancel()
			fmt.Println("Exchange() stop receiving")
			break
		}

		fmt.Println("<<<", in.T.AsTime(), in.Text)
	}
	return nil
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
