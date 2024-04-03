package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/moonspirit/grpc-tracing-bench/pkg/proto/echopb"
	"github.com/moonspirit/grpc-tracing-bench/pkg/telemetry"
	"github.com/moonspirit/grpc-tracing-bench/pkg/xmd"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	port = 8000
)

type server struct {
	echopb.UnimplementedEchoServiceServer
}

func (s *server) Echo(ctx context.Context, req *echopb.Request) (*echopb.Response, error) {
	md, _ := xmd.FromContext(ctx)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("x-id", md.Get("x-id")[0]),
		attribute.String("x-openid", md.Get("x-openid")[0]),
		attribute.String("x-uid", md.Get("x-uid")[0]))
	return &echopb.Response{
		Msg: req.Msg,
		Id:  req.Id,
	}, nil
}

func main() {
	flag.Parse()

	// tracing exporter
	tracingExporter, _ := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint("127.0.0.1:15231"),
		otlptracegrpc.WithCompressor("gzip"),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(4194304))),
	)

	sampler := telemetry.NewSampler(1.0 / 1024)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(tracingExporter)),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// listen pprof
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.NumStreamWorkers(1000), // avoid frequent goroutine creation and stack growth
		grpc.ReadBufferSize(1024*1024), grpc.WriteBufferSize(1024*1024),
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor()))
	echopb.RegisterEchoServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
