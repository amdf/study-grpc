package main

import (
	"context"
	"log"
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

	r, err := c.SimpleFunction(ctx, &pb.SimpleQuery{Text: "SoMeTeXt"})
	if err != nil {
		log.Fatalln("fail to call SimpleFunction", err)
	}
	i := r.GetRuneCount()
	log.Println("Rune count:", i)
}
