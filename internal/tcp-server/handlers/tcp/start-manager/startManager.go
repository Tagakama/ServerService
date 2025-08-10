package startManager

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/config"
	"net"
)

func New(config *config.Config) (net.Listener, error) {
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%s", config.Address, config.Port))
	if err != nil {
		fmt.Sprintf("Server not listen %v", err)
	}
	fmt.Println("Server is listening on " + config.Port)
	return server, err
}
