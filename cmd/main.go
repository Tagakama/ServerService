package main

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/config"
	handlers "github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/handleConnection"
	"github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/startManager"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
)

func main() {
	cfg := config.MustLoad()
	workerPool := workers.NewWorkerPool(cfg.WorkerCount)

	serverManagerListener, err := startManager.CreateServerManager(cfg)
	if err != nil {
		fmt.Printf("Error creating server manager: %v", err)
	}
	defer serverManagerListener.Close()

	for {
		conn, err := serverManagerListener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handlers.HandleConnection(conn, workerPool)
	}

}
