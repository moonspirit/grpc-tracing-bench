package xmd

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type CtxMetadataCacheKey struct{}

func AddContext(ctx context.Context, md metadata.MD) context.Context {
	return context.WithValue(ctx, CtxMetadataCacheKey{}, md)
}

func FromContext(ctx context.Context) (metadata.MD, bool) {
	md, ok := ctx.Value(CtxMetadataCacheKey{}).(metadata.MD)
	if ok {
		return md, true
	}
	md, ok = metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}
	return md, true
}
