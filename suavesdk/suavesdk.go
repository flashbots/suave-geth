package suavesdk

import (
	"context"
	"net"

	"github.com/ethereum/go-ethereum/suavesdk/proto"
	"google.golang.org/grpc"
)

type Suapp struct {
	proto.UnimplementedSuaveServer

	dispatchTable *DispatchTable
}

type Config struct {
	fn interface{}
}

type Option func(*Config)

func WithFunction(fn interface{}) Option {
	return func(c *Config) {
		c.fn = fn
	}
}

func NewSuapp(opts ...Option) *Suapp {
	c := &Config{}
	for _, o := range opts {
		o(c)
	}
	s := &Suapp{
		dispatchTable: NewDispatchTable(),
	}
	if c.fn != nil {
		s.dispatchTable.MustRegister(c.fn)
	}

	// start grpc server
	s.startGrpcServer()

	return s
}

func (s *Suapp) startGrpcServer() {
	rpc := grpc.NewServer()
	proto.RegisterSuaveServer(rpc, s)

	// start server with grpc
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	rpc.Serve(lis)
}

func (s *Suapp) NewTopic(name string) *Topic {
	return &Topic{}
}

// TODO
type Topic struct {
}

func (t *Topic) Publish(data []byte) {

}

func (t *Topic) Subscribe(func(data []byte)) {

}

// ** TRANSPORT ENDPOINTS **

func (s *Suapp) Call(ctx context.Context, req *proto.CallRequest) (*proto.CallResponse, error) {
	out, err := s.dispatchTable.Run(req.Input)
	if err != nil {
		return nil, err
	}
	return &proto.CallResponse{Output: out}, nil
}
