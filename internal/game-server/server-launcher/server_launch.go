package server_launcher

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/config"
	"github.com/Tagakama/ServerManager/internal/matchmaking/room"
	"net"
	"os/exec"
	"strconv"
)

type Launcher interface {
	LaunchGameServer(settings *room.Room)
}

type ServerLauncher struct {
	versionPath string
	execName    string
}

func New(cfg *config.Config) *ServerLauncher {
	return &ServerLauncher{
		versionPath: cfg.VersionPath,
		execName:    cfg.ExecutableName,
	}
}

func (s *ServerLauncher) LaunchGameServer(settings *room.Room) {
	port, tcpListener, err := FindFreePort()
	if err != nil {
		panic(err)
	}

	logFilePath := fmt.Sprintf("Logs/Room_%d.log", settings.ID)
	unicName := fmt.Sprintf("%s%s%d", settings.AppVersion, settings.CurrentMap, settings.ID)
	cmd := exec.Command(s.versionPath+settings.AppVersion+s.execName,
		"-nographics", "-dedicatedServer", "-batchmode", "-fps", "60", "-dfill", "-UserID", unicName,
		"-sessionName", unicName, "-logFile", logFilePath,
		"-port", strconv.Itoa(port), "-region eu",
		"-serverName", unicName, "-scene", settings.CurrentMap)

	tcpListener.Close()

	err = cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start server %d: %v\n", settings.ID, err)
		return
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Server %d stopped with error: %v\n", settings.ID, err)
		}
	}()

	fmt.Printf("Server %s started.\n", unicName)
}

func FindFreePort() (int, *net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, nil, err
	}
	return listener.Addr().(*net.TCPAddr).Port, listener, nil
}
