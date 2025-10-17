package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that instruments requests with Prometheus metrics
func UnaryServerInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		method := info.FullMethod

		// Increment active requests
		metrics.GRPCActiveRequests.WithLabelValues(method).Inc()
		defer metrics.GRPCActiveRequests.WithLabelValues(method).Dec()

		// Start timer
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Record metrics
		metrics.GRPCRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that instruments requests with Prometheus metrics
func StreamServerInterceptor(metrics *Metrics) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		method := info.FullMethod

		// Increment active requests
		metrics.GRPCActiveRequests.WithLabelValues(method).Inc()
		defer metrics.GRPCActiveRequests.WithLabelValues(method).Dec()

		// Start timer
		start := time.Now()

		// Call handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Record metrics
		metrics.GRPCRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())

		return err
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor for outgoing requests
func UnaryClientInterceptor(metrics *Metrics) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Start timer
		start := time.Now()

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Record metrics
		metrics.GRPCRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())

		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor for outgoing requests
func StreamClientInterceptor(metrics *Metrics) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Start timer
		start := time.Now()

		// Call streamer
		stream, err := streamer(ctx, desc, cc, method, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Record metrics
		metrics.GRPCRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())

		return stream, err
	}
}