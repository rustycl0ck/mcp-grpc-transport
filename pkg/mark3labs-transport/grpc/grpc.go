package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	pb "github.com/rustycl0ck/mcp-grpc-transport/pkg/protogen/jsonrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/structpb"
)

// GrpcServerTransport implements server-side transport for grpc communication
type GrpcServer struct {
	pb.UnimplementedJSONRPCServiceServer
	mcpserver *mcpsrv.MCPServer
	// onMessage func(ctx context.Context, message *transport.BaseJsonRpcMessage)
	host     string
	port     int
	grpcOpts []grpc.ServerOption
}

type GrpcServerOption func(*GrpcServer)

func WithHost(host string) GrpcServerOption {
	return func(s *GrpcServer) {
		s.host = host
	}
}

func WithPort(port int) GrpcServerOption {
	return func(s *GrpcServer) {
		s.port = port
	}
}

func WithGrpcOpts(opts ...grpc.ServerOption) GrpcServerOption {
	return func(s *GrpcServer) {
		s.grpcOpts = opts
	}
}

// NewGrpcServer creates a new MCP Server with gRPC Transport
func NewGrpcServer(server *mcpsrv.MCPServer, opts ...GrpcServerOption) *GrpcServer {
	srv := &GrpcServer{
		port:      50051,
		mcpserver: server,
	}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

type ctxKey string

func (t *GrpcServer) Listen(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", t.host, t.port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(t.grpcOpts...)
	reflection.Register(grpcServer)

	pb.RegisterJSONRPCServiceServer(grpcServer, t)

	return grpcServer.Serve(lis)
}

func (t *GrpcServer) Close() error {
	return nil
}

func (g *GrpcServer) Transport(stream pb.JSONRPCService_TransportServer) error {
	fmt.Printf("transport started...\n")

	ctx := context.WithValue(context.Background(), ctxKey("stream"), stream)

	for {
		ms, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Stream closed by client")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("Received message...")
		// TODO: debug log the recevied request
		// fmt.Printf("Received message...: %s\n", ms)

		if baseMsg, err := ToJsonRpcMessage(ms); err != nil {
			return err
		} else {
			jmsg := g.mcpserver.HandleMessage(ctx, baseMsg)
			pbmsg, err := FromJsonRpcMessage(jmsg, ms.TypedId)
			if err != nil {
				return err
			}
			// TODO: debug log the response
			// fmt.Printf("Sending message...: %s\n", pbmsg)

			if er := stream.Send(pbmsg); er != nil {
				return er
			}
		}
	}

	return nil
}

func FromJsonRpcMessage(m mcp.JSONRPCMessage, id *pb.ID) (*pb.GenericJSONRPCMessage, error) {
	if m == nil {
		return nil, fmt.Errorf("input message is nil")
	}

	msg := &pb.GenericJSONRPCMessage{}

	switch v := m.(type) {
	case mcp.JSONRPCRequest:
		msg.Jsonrpc = v.JSONRPC
		msg.TypedId = id
		msg.Method = v.Request.Method
		params, err := genericStruct(v.Params)
		if err != nil {
			return nil, err
		}
		msg.Params = params

	case mcp.JSONRPCResponse:
		msg.Jsonrpc = v.JSONRPC
		msg.TypedId = id
		result, err := genericStruct(v.Result)
		if err != nil {
			return nil, err
		}
		msg.Result = result

	case mcp.JSONRPCError:
		msg.Jsonrpc = v.JSONRPC
		msg.TypedId = id
		msg.Error = &pb.JSONRPCError{
			Code:    int32(v.Error.Code),
			Message: v.Error.Message,
			// Data:    v.Error.Data,
		}

	case mcp.JSONRPCNotification:
		msg.Jsonrpc = v.JSONRPC
		msg.Method = v.Notification.Method
		params, err := genericStruct(v.Notification.Params)
		if err != nil {
			return nil, err
		}
		msg.Params = params

	default:
		return nil, fmt.Errorf("unsupported MCP message type: %T", m)
	}

	return msg, nil
}

func ToJsonRpcMessage(m *pb.GenericJSONRPCMessage) (json.RawMessage, error) {
	if m == nil {
		return nil, fmt.Errorf("input message is nil")
	}

	switch {
	case m.TypedId != nil && m.Method != "":
		// JSON-RPC Request
		tmp := mcp.JSONRPCRequest{
			JSONRPC: m.Jsonrpc,
			ID:      mcp.NewRequestId(m.TypedId),
			Params:  m.Params,
			Request: mcp.Request{
				Method: m.Method,
				Params: mcp.RequestParams{Meta: nil}, //TODO: Fix this to pass correct value
			},
		}
		return marshalToRawMessage(tmp)

	case m.TypedId != nil && m.Result != nil:
		// JSON-RPC Response
		tmp := mcp.JSONRPCResponse{
			JSONRPC: m.Jsonrpc,
			ID:      mcp.NewRequestId(m.TypedId),
			Result:  m.Result,
		}
		return marshalToRawMessage(tmp)

	case m.TypedId != nil && m.Error != nil:
		// JSON-RPC Error
		tmp := mcp.NewJSONRPCError(
			mcp.NewRequestId(m.TypedId),
			int(m.Error.Code),
			m.Error.Message,
			m.Error.Data,
		)
		return marshalToRawMessage(tmp)

	case m.TypedId == nil && m.Method != "":
		// JSON-RPC Notification
		tmp := mcp.JSONRPCNotification{
			JSONRPC: m.Jsonrpc,
			Notification: mcp.Notification{
				Method: m.Method,
				Params: mcp.NotificationParams{}, // TODO: Implement this correctly
			},
		}
		return marshalToRawMessage(tmp)

	default:
		return nil, fmt.Errorf("failed to determine the type of the message")
	}
}

// Helper function to marshal to json.RawMessage
func marshalToRawMessage(v any) (json.RawMessage, error) {
	rawBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(rawBytes), nil
}

// Helper function to convert any interface to structpb.Struct
func genericStruct(v any) (*structpb.Struct, error) {
	rawBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(rawBytes, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}
