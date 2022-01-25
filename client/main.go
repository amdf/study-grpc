package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"

	pb "github.com/amdf/study-grpc/svc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SimpleService struct {
	ClientGRPC pb.SimpleServiceClient
}

func (client *SimpleService) SimpleFunction(text string) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := client.ClientGRPC.SimpleFunction(ctx, &pb.SimpleQuery{Text: text})
	if err != nil {
		log.Fatalln("fail to call SimpleFunction()", err)
	}
	i := r.GetRuneCount()
	fmt.Println("I. Rune count:", i)
}

func (client *SimpleService) Sum(num int) {
	stream, err := client.ClientGRPC.Sum(context.Background())
	if err != nil {
		log.Fatalln("fail to call Sum()", err)
	}
	for i := 0; i < rand.Intn(num); i++ {
		var n pb.Number
		n.Value = uint32(rand.Int31n(25))
		err = stream.Send(&n)
		if err != nil {
			log.Fatalln("fail to send Sum() argument", err)
		}
	}
	result, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalln("error get Sum() result", err)
	}
	fmt.Println("II: Sum result:", result.Value)
}

func (client *SimpleService) GenerateWords(count int32, delay time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	stream, err := client.ClientGRPC.GenerateWords(ctx, &pb.WantWords{Count: count, Delay: durationpb.New(delay)})
	if err != nil {
		return
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("error reading GenerateWords()", err)
			break
		}
		fmt.Println("III: word", in.T.AsTime(), in.Text)
	}
}

func (client *SimpleService) Exchange() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.ClientGRPC.Exchange(ctx)
	if nil != err {
		return
	}
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
				text := "from client"
				if nil != stream.Send(&pb.SomeText{T: t, Text: text}) {
					stop = true
				}
				fmt.Println(">>>", t.AsTime().Local().Format("15:04:05"), text)
				ticker.Reset(time.Millisecond * 100 * time.Duration(rand.Intn(8)))
			}
		}

		log.Println("Exchange() stop sending")
	}(ctx)

	ipaddr := "(unknown)"
	p, ok := peer.FromContext(stream.Context())
	if ok {
		ipaddr = p.Addr.String()
	}

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
}

func timingInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	fmt.Printf(`--
		call=%v
		req=%#v
		reply=%#v
		time=%v
		err=%v
		`, method, req, reply, time.Since(start), err)
	return err
}

func NewSimpleService() (s *SimpleService, err error) {

	conn, err := grpc.Dial("[::]:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(timingInterceptor),
	)
	if err != nil {
		return
	}

	//TODO: close conn somewhere?

	c := pb.NewSimpleServiceClient(conn)
	s = &SimpleService{ClientGRPC: c}
	return
}

func main() {

	fmt.Println()

	client, err := NewSimpleService()
	if err != nil {
		log.Fatalln("fail to connect", err)
	}
	//simple method:
	client.SimpleFunction("SoMeTeXt")

	//method with argument stream:
	client.Sum(20)

	//method with result stream:
	client.GenerateWords(10, time.Second/4)

	//bidirectional stream:
	client.Exchange()
}
