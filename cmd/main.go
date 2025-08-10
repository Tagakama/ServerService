package main

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/config"
	server_launcher "github.com/Tagakama/ServerManager/internal/game-server/server-launcher"
	"github.com/Tagakama/ServerManager/internal/matchmaking/matchmaker"
	handlers "github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/handle-connection"
	"github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/start-manager"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
)

func main() {
	cfg := config.MustLoad()
	serverLauncher := server_launcher.New(cfg)
	newMatchmaker := matchmaker.New(serverLauncher)

	workerPool := workers.NewWorkerPool(cfg.WorkerCount, newMatchmaker)

	serverManager, err := startManager.New(cfg)
	if err != nil {
		panic(err)
	}

	defer serverManager.Close()

	for {
		conn, err := serverManager.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handlers.HandleConnection(conn, workerPool)
	}

}
