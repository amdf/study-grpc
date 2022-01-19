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
	stream, err := c.GenerateWords(ctx, &pb.WantWords{Count: 10, Delay: durationpb.New(time.Second)})
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
}
