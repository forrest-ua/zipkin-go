// +build go1.9

package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/openzipkin/zipkin-go/model"
	service "github.com/openzipkin/zipkin-go/proto/testing"
)

var (
	server     *grpc.Server
	serverAddr string
)

func TestGrpc(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Grpc Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	lis, err := net.Listen("tcp", ":0")
	gomega.Expect(lis, err).ToNot(gomega.BeNil(), "failed to listen to tcp port")

	server = grpc.NewServer()
	service.RegisterHelloServiceServer(server, &TestHelloService{})
	go func() {
		_ = server.Serve(lis)
	}()
	serverAddr = lis.Addr().String()
})

var _ = ginkgo.AfterSuite(func() {
	server.Stop()
})

type sequentialIdGenerator struct {
	nextTraceId uint64
	nextSpanId  uint64
}

func newSequentialIdGenerator() *sequentialIdGenerator {
	return &sequentialIdGenerator{1, 1}
}

func (g *sequentialIdGenerator) SpanID(traceID model.TraceID) model.ID {
	id := model.ID(g.nextSpanId)
	g.nextSpanId++
	return id
}

func (g *sequentialIdGenerator) TraceID() model.TraceID {
	id := model.TraceID{
		High: 0,
		Low:  g.nextTraceId,
	}
	g.nextTraceId++
	return id
}

type TestHelloService struct{}

func (s *TestHelloService) Hello(ctx context.Context, req *service.HelloRequest) (*service.HelloResponse, error) {
	if req.Payload == "fail" {
		return nil, status.Error(codes.Aborted, "fail")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not parse incoming metadata")
	}

	resp := &service.HelloResponse{
		Payload:  "World",
		Metadata: map[string]string{},
	}

	for k := range md {
		// Just append the first value for a key for simplicity since we don't use multi-value headers.
		resp.GetMetadata()[k] = md[k][0]
	}

	return resp, nil
}
