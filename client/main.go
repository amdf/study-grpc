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
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CallSimpleFunction(c pb.SimpleServiceClient) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SimpleFunction(ctx, &pb.SimpleQuery{Text: "SoMeTeXt"})
	if err != nil {
		log.Fatalln("fail to call SimpleFunction()", err)
	}
	i := r.GetRuneCount()
	fmt.Println("I. Rune count:", i)
}

func CallSum(c pb.SimpleServiceClient) {
	stream, err := c.Sum(context.Background())
	if err != nil {
		log.Fatalln("fail to call Sum()", err)
	}
	for i := 0; i < rand.Intn(20); i++ {
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

func CallGenerateWords(c pb.SimpleServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	stream, err := c.GenerateWords(ctx, &pb.WantWords{Count: 10, Delay: durationpb.New(time.Second / 4)})
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

func CallExchange(c pb.SimpleServiceClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := c.Exchange(ctx)
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
				fmt.Println(">>>", t.AsTime(), text)
				ticker.Reset(time.Millisecond * 100 * time.Duration(rand.Intn(8)))
			}
		}

		log.Println("Exchange() stop sending")
	}(ctx)
	for {
		in, err := stream.Recv()
		//if err == io.EOF {
		if err != nil {
			cancel()
			fmt.Println("Exchange() stop receiving")
			break
		}

		fmt.Println("<<<", in.T.AsTime(), in.Text)
	}
}

func main() {
	conn, err := grpc.Dial("[::]:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalln("fail to connect", err)
	}
	defer conn.Close()

	c := pb.NewSimpleServiceClient(conn)

	fmt.Println()

	//simple method:
	CallSimpleFunction(c)

	//method with argument stream:
	CallSum(c)

	//method with result stream:
	CallGenerateWords(c)

	//bidirectional stream:
	CallExchange(c)
}
