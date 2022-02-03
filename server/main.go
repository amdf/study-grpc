package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	pb "github.com/amdf/study-grpc/svc"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/time/rate"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/tap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	SVCRPCADDR  = "localhost:50051"
	SVCHTTPADDR = "localhost:8081" //"0.0.0.0:80"
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

	count := len(q.NewText)
	result = &pb.SimpleResponse{RuneCount: int32(count)}

	fmt.Println("SimpleFunction. NewText = ", q.NewText, "Time =", q.Time.AsTime().Local())

	return
}

func (server *SimpleServiceServer) Image(context.Context, *empty.Empty) (body *httpbody.HttpBody, err error) {
	var buf bytes.Buffer

	bg := image.Black
	rgba := image.NewRGBA(image.Rect(0, 0, 300, 200))
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, draw.Src)

	err = png.Encode(&buf, rgba)

	body = &httpbody.HttpBody{
		ContentType: "image/png",
		Data:        buf.Bytes(),
	}

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

var m map[string]*rate.Limiter

func rateLimiter(ctx context.Context, info *tap.Info) (context.Context, error) {
	user := getAddr(ctx)
	if m[user] == nil {
		// NewLimiter returns a new Limiter that allows events up to rate r and permits
		// bursts of at most b tokens.
		m[user] = rate.NewLimiter(1, 1) // QPS, burst
	}

	// Allow indicates whether one event may happen at time.Now().
	if !m[user].Allow() {
		return nil, status.Errorf(codes.ResourceExhausted,
			"client exceeded rate limit")
	}
	return ctx, nil
}

func runGateway() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := pb.RegisterSimpleServiceHandlerFromEndpoint(ctx, mux, SVCRPCADDR, opts)
	if err != nil {
		log.Fatalln("fail to register http", err)
	}

	log.Println("starting http server at ", SVCHTTPADDR)
	// Start HTTP server (and proxy calls to gRPC server endpoint)

	if err != http.ListenAndServe(SVCHTTPADDR, mux) {
		log.Fatalln("fail to serve http", err)
	}
}

func main() {
	m = make(map[string]*rate.Limiter)
	var server SimpleServiceServer
	lis, err := net.Listen("tcp", SVCRPCADDR)
	if err != nil {
		log.Fatalln("fail to listen", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
		grpc.InTapHandle(rateLimiter),
	)
	pb.RegisterSimpleServiceServer(s, &server)

	log.Println("starting server at ", lis.Addr())

	go runGateway()

	err = s.Serve(lis)
	if err != nil {
		log.Fatalln("fail to server", err)
	}
}
