# mcp-grpc-transport

An MCP transport which uses gRPC for communication between the client and the MCP server. This will make it easier to host MCP servers on remote instances just like any other web servers.

Usage of gRPC as the transport, allows to reuse any existing infrastructure and processes for authentication and authorization.

> [!NOTE]
> Supports only the two most widely used libraries currently:
> - [metoro-io/mcp-golang](https://github.com/metoro-io/mcp-golang) - Upstream [PR #118](https://github.com/metoro-io/mcp-golang/pull/118)
> - [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)

## Usage


#### [`examples/metoro-io-server`](./examples/metoro-io-server)

In the [`metoro-io/mcp-golang` server example](https://github.com/metoro-io/mcp-golang?tab=readme-ov-file#server-example), just replace the transport as follows:

```diff
+ import grpctransport "github.com/rustycl0ck/mcp-grpc-transport/pkg/metoro-io-transport/grpc"
  
  func main() {
      ...
  
-     server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
+     server := mcp_golang.NewServer(grpctransport.NewGrpcServerTransport())
      ...
  }
```

You can customize the server with more options if required:
```go
import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main(){
	serverTransport := grpctransport.NewGrpcServerTransport(
		grpctransport.WithPort(10051),
		grpctransport.WithGrpcOpts(
			grpc.UnaryInterceptor(loggingInterceptor),
			grpc.MaxRecvMsgSize(1024*1024),        // 1MB
			grpc.Creds(insecure.NewCredentials()), // TODO: Use TLS in production!
		),
	)

	// Create a new server with the transport
	server := mcp_golang.NewServer(serverTransport)
}
```

#### [`examples/mark3labs-server`](./examples/mark3labs-server)

In the [`mark3labs/mcp-go` server example](https://github.com/mark3labs/mcp-go?tab=readme-ov-file#quickstart), just replace the transport as follows:

```diff
+ import grpctransport "github.com/rustycl0ck/mcp-grpc-transport/pkg/mark3labs-transport/grpc"

  func main() {
      ...

      // Start the server
-     if err := server.ServeStdio(s); err != nil {
-         fmt.Printf("Server error: %v\n", err)
-     }

+     srv := grpctransport.NewGrpcServer(s)
+     if err := srv.Listen(context.Background()); err != nil {
+         fmt.Printf("Server error: %v\n", err)
+     }
      ...
  }
```

You can customize the server with more options if required:
```go
import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main(){
-     srv := grpctransport.NewGrpcServer(s)
+     srv := grpctransport.NewGrpcServer(s,
	  	grpctransport.WithPort(10051),
	  	grpctransport.WithGrpcOpts(
	  		grpc.UnaryInterceptor(loggingInterceptor),
	  		grpc.MaxRecvMsgSize(1024*1024),        // 1MB
	  		grpc.Creds(insecure.NewCredentials()), // TODO: Use TLS in production!
	  	),
	  )
}
```

## Example

Start the server:
```console
$ go run examples/metoro-io-server.go
transport started...
Received message...
transport started...
Received message...

```

Configure your IDE client (following confugration is for Cursor IDE's `mcp.json`)
```json
{
  "mcpServers": {
    "my-mcp-server": {
      "command": "go",
      "args": [
        "run",
        "github.com/rustycl0ck/mcp-grpc-transport/cmd/client@latest",
        "--address",
        "localhost:50051"  // Replace with actual server location if hosted remotely
      ]
    }
  }
}
```

Or test the client locally directly through CLI:
```console
$ echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | go run github.com/rustycl0ck/mcp-grpc-transport/cmd/client@latest
{"id":1,"jsonrpc":"2.0","result":{"tools":[{"description":"Get the weather forecast for temperature, wind speed and relative humidity","inputSchema":{"$schema":"https://json-schema.org/draft/2020-12/schema","properties":{"latitude":{"description":"The latitude of the location to get the weather for","type":"number"},"longitude":{"description":"The longitude of the location to get the weather for","type":"number"}},"required":["longitude","latitude"],"type":"object"},"name":"get_weather"},{"description":"Says hello","inputSchema":{"$schema":"https://json-schema.org/draft/2020-12/schema","properties":{"name":{"description":"The name to say hello to","type":"string"}},"required":["name"],"type":"object"},"name":"hello"}]}}
```

## License
[MIT](LICENSE)
