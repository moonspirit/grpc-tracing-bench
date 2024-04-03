// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/moonspirit/grpc-tracing-bench/pkg/xmd"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	// GRPCStatusCodeKey is convention for numeric status code of a gRPC request.
	GRPCStatusCodeKey = attribute.Key("rpc.grpc.status_code")
	// Version is the current release version of the gRPC instrumentation.
	Version = "0.46.1"
)

// Semantic conventions for attribute keys for gRPC.
const (
	// Type of message transmitted or received.
	RPCMessageTypeKey = attribute.Key("message.type")

	// Identifier of message transmitted or received.
	RPCMessageIDKey = attribute.Key("message.id")
)

// Semantic conventions for common RPC attributes.
var (
	// Semantic convention for gRPC as the remoting system.
	RPCSystemGRPC = semconv.RPCSystemGRPC

	// Semantic conventions for RPC message types.
	RPCMessageTypeSent     = RPCMessageTypeKey.String("SENT")
	RPCMessageTypeReceived = RPCMessageTypeKey.String("RECEIVED")
)

// InterceptorType is the flag to define which gRPC interceptor
// the InterceptorInfo object is.
type InterceptorType uint8

const (
	// UndefinedInterceptor is the type for the interceptor information that is not
	// well initialized or categorized to other types.
	UndefinedInterceptor InterceptorType = iota
	// UnaryClient is the type for grpc.UnaryClient interceptor.
	UnaryClient
	// StreamClient is the type for grpc.StreamClient interceptor.
	StreamClient
	// UnaryServer is the type for grpc.UnaryServer interceptor.
	UnaryServer
	// StreamServer is the type for grpc.StreamServer interceptor.
	StreamServer
)

// InterceptorInfo is the union of some arguments to four types of
// gRPC interceptors.
type InterceptorInfo struct {
	// Method is method name registered to UnaryClient and StreamClient
	Method string
	// UnaryServerInfo is the metadata for UnaryServer
	UnaryServerInfo *grpc.UnaryServerInfo
	// StreamServerInfo if the metadata for StreamServer
	StreamServerInfo *grpc.StreamServerInfo
	// Type is the type for interceptor
	Type InterceptorType
}

// Filter is a predicate used to determine whether a given request in
// interceptor info should be traced. A Filter must return true if
// the request should be traced.
type Filter func(*InterceptorInfo) bool

// config is a group of options for this instrumentation.
type config struct {
	Filter         Filter
	Propagators    propagation.TextMapPropagator
	TracerProvider trace.TracerProvider

	ReceivedEvent bool
	SentEvent     bool

	tracer trace.Tracer
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option) *config {
	c := &config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
	}
	for _, o := range opts {
		o.apply(c)
	}

	c.tracer = c.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version),
	)

	return c
}

type messageType attribute.KeyValue

// Event adds an event of the messageType to the span associated with the
// passed context with a message id.
func (m messageType) Event(ctx context.Context, id int, _ interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.AddEvent("message", trace.WithAttributes(
		attribute.KeyValue(m),
		RPCMessageIDKey.Int(id),
	))
}

var (
	messageSent     = messageType(RPCMessageTypeSent)
	messageReceived = messageType(RPCMessageTypeReceived)
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	cfg := newConfig(nil)
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version),
	)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		i := &InterceptorInfo{
			UnaryServerInfo: info,
			Type:            UnaryServer,
		}
		if cfg.Filter != nil && !cfg.Filter(i) {
			return handler(ctx, req)
		}

		ctx = extract(ctx, cfg.Propagators)
		name, attr := telemetryAttributes(info.FullMethod, peerFromCtx(ctx))

		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		if cfg.ReceivedEvent {
			messageReceived.Event(ctx, 1, req)
		}

		resp, err := handler(ctx, req)

		s, _ := status.FromError(err)
		if err != nil {
			statusCode, msg := serverStatus(s)
			span.SetStatus(statusCode, msg)
			if cfg.SentEvent {
				messageSent.Event(ctx, 1, s.Proto())
			}
		} else {
			if cfg.SentEvent {
				messageSent.Event(ctx, 1, resp)
			}
		}
		grpcStatusCodeAttr := statusCodeAttr(s.Code())
		span.SetAttributes(grpcStatusCodeAttr)

		return resp, err
	}
}

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	cfg := newConfig(nil)
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version),
	)

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		callOpts ...grpc.CallOption,
	) error {
		i := &InterceptorInfo{
			Method: method,
			Type:   UnaryClient,
		}
		if cfg.Filter != nil && !cfg.Filter(i) {
			return invoker(ctx, method, req, reply, cc, callOpts...)
		}

		name, attr := telemetryAttributes(method, cc.Target())

		ctx, span := tracer.Start(
			ctx,
			name,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		ctx = inject(ctx, cfg.Propagators)

		if cfg.SentEvent {
			messageSent.Event(ctx, 1, req)
		}

		err := invoker(ctx, method, req, reply, cc, callOpts...)

		if cfg.ReceivedEvent {
			messageReceived.Event(ctx, 1, reply)
		}

		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(statusCodeAttr(s.Code()))
		} else {
			span.SetAttributes(statusCodeAttr(grpc_codes.OK))
		}

		return err
	}
}

type metadataSupplier struct {
	metadata *metadata.MD
}

// assert that metadataSupplier implements the TextMapCarrier interface.
var _ propagation.TextMapCarrier = &metadataSupplier{}

func (s *metadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s *metadataSupplier) Set(key string, value string) {
	s.metadata.Set(key, value)
}

func (s *metadataSupplier) Keys() []string {
	out := make([]string, 0, len(*s.metadata))
	for key := range *s.metadata {
		out = append(out, key)
	}
	return out
}

// telemetryAttributes returns a span name and span and metric attributes from
// the gRPC method and peer address.
func telemetryAttributes(fullMethod, peerAddress string) (string, []attribute.KeyValue) {
	name, service, method := parseFullMethod(fullMethod)

	attrs := make([]attribute.KeyValue, 0, 5)
	attrs = append(attrs, RPCSystemGRPC)
	if service != "" {
		attrs = append(attrs, semconv.RPCService(service))
	}
	if method != "" {
		attrs = append(attrs, semconv.RPCMethod(method))
	}

	host, p, _ := net.SplitHostPort(peerAddress)
	if host == "" {
		host = "127.0.0.1"
	}
	attrs = append(attrs, semconv.NetPeerName(host))
	port, _ := strconv.Atoi(p)
	attrs = append(attrs, semconv.NetPeerPort(port))
	return name, attrs
}

// peerFromCtx returns a peer address from a context, if one exists.
func peerFromCtx(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

// statusCodeAttr returns status code attribute based on given gRPC code.
func statusCodeAttr(c grpc_codes.Code) attribute.KeyValue {
	return GRPCStatusCodeKey.Int64(int64(c))
}

func inject(ctx context.Context, propagators propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	propagators.Inject(ctx, &metadataSupplier{
		metadata: &md,
	})
	return metadata.NewOutgoingContext(ctx, md)
}

func extract(ctx context.Context, propagators propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	return propagators.Extract(xmd.AddContext(ctx, md), &metadataSupplier{
		metadata: &md,
	})
}

// serverStatus returns a span status code and message for a given gRPC
// status code. It maps specific gRPC status codes to a corresponding span
// status code and message. This function is intended for use on the server
// side of a gRPC connection.
//
// If the gRPC status code is Unknown, DeadlineExceeded, Unimplemented,
// Internal, Unavailable, or DataLoss, it returns a span status code of Error
// and the message from the gRPC status. Otherwise, it returns a span status
// code of Unset and an empty message.
func serverStatus(grpcStatus *status.Status) (codes.Code, string) {
	switch grpcStatus.Code() {
	case grpc_codes.Unknown,
		grpc_codes.DeadlineExceeded,
		grpc_codes.Unimplemented,
		grpc_codes.Internal,
		grpc_codes.Unavailable,
		grpc_codes.DataLoss:
		return codes.Error, grpcStatus.Message()
	default:
		return codes.Unset, ""
	}
}

// parseFullMethod returns a span name following the OpenTelemetry semantic
// conventions as well as all applicable span attribute.KeyValue attributes based
// on a gRPC's FullMethod.
//
// Parsing is consistent with grpc-go implementation:
// https://github.com/grpc/grpc-go/blob/v1.57.0/internal/grpcutil/method.go#L26-L39
func parseFullMethod(fullMethod string) (name string, service string, method string) {
	if !strings.HasPrefix(fullMethod, "/") {
		// Invalid format, does not follow `/package.service/method`.
		return fullMethod, "", ""
	}
	name = fullMethod[1:]
	pos := strings.LastIndex(name, "/")
	if pos < 0 {
		// Invalid format, does not follow `/package.service/method`.
		return name, "", ""
	}
	return name, name[:pos], name[pos+1:]
}
