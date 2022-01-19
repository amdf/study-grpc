package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	pb "github.com/amdf/study-grpc/svc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("[::]:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalln("fail to connect", err)
	}
	defer conn.Close()

	c := pb.NewSimpleServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//simple method:

	r, err := c.SimpleFunction(ctx, &pb.SimpleQuery{Text: "SoMeTeXt"})
	if err != nil {
		log.Fatalln("fail to call SimpleFunction()", err)
	}
	i := r.GetRuneCount()
	log.Println("I. Rune count:", i)

	//method with argument stream:
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

	//method with result stream:
	// ctx2, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// c.GenerateWords(ctx2, &pb.WantWords{Count: 10, Delay: durationpb.New(time.Microsecond * 100)})
}
