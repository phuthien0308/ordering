package clients

import (
	"os"

	"github.com/phuthien0308/ordering/config/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var configAddress = "localhost:50051"
var ConfigClient = func() pb.ConfigClient {
	cfAddr := os.Getenv("CONF_SERVER")
	if len(cfAddr) > 0 {
		configAddress = cfAddr
	}
	conn, err := grpc.NewClient(configAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	return pb.NewConfigClient(conn)
}()
