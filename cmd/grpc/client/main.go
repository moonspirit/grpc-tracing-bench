package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/moonspirit/grpc-tracing-bench/pkg/metrics"
	"github.com/moonspirit/grpc-tracing-bench/pkg/proto/echopb"
	"github.com/moonspirit/grpc-tracing-bench/pkg/telemetry"
	"github.com/moonspirit/grpc-tracing-bench/pkg/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	serverAddr         = flag.String("addr", "localhost:8000", "server address for benchmark")
	concurrecy         = flag.Int("concurrency", 1, "concurrent client goroutine for benchmark")
	connectionPoolSize = flag.Int("pool", 1, "connection pool size for benchmark")
	payloadSize        = flag.Int("payload", 1024, "payload size for benchmark")
)

var summary = metrics.NewLatencyStats()

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

	var idgen atomic.Int64

	conns := make([]*grpc.ClientConn, 0, *connectionPoolSize)
	for i := 0; i < *connectionPoolSize; i++ {
		conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithReadBufferSize(1024*1024), grpc.WithWriteBufferSize(1024*1024),
			grpc.WithUnaryInterceptor(telemetry.UnaryClientInterceptor()))
		if err != nil {
			panic(err)
		}
		conns = append(conns, conn)
	}

	gidBase := int64(10001)

	for i := 0; i < *concurrecy; i++ {
		go func(c *grpc.ClientConn, i int64) {
			cc := echopb.NewEchoServiceClient(c)
			for {
				now := time.Now()
				id := idgen.Add(1)

				gid := gidBase + i
				xgid := strconv.FormatInt(gid, 10)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				ctx = metadata.AppendToOutgoingContext(ctx, "x-id", xgid, "x-openid", xgid, "x-uid", xgid)

				resp, err := cc.Echo(ctx, &echopb.Request{
					Msg: string(util.RandomBytes(*payloadSize)),
					Id:  id,
				})
				cancel()
				if err != nil {
					panic(err)
				}
				if resp.Id != id {
					panic("id not match")
				}
				summary.Observe(time.Since(now))
			}
		}(conns[i%len(conns)], int64(i))
	}

	for {
		time.Sleep(time.Second)
		log.Println("requests: ", summary.String())
	}
}
