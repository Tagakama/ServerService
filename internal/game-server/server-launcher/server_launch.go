package server_launcher

import (
	"fmt"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func launchGameServer(settings _type.RoomSettings, connection []*_type.PendingConnection) {
	port, err := FindFreePort()
	if err != nil {
		fmt.Sprintf("Failed to find free port: %v", err)
	}
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}

	logFilePath := fmt.Sprintf("Logs/Room_%d.log", settings.ID)

	servisePath := fmt.Sprintf("%s/%s", filepath.Dir(exePath))
	cmd := exec.Command(cfg.LocalStorage.Directory+server.AppVersion+cfg.LocalStorage.Name,
		"-nographics", "-dedicatedServer", "-batchmode", "-fps", "60", "-dfill", "-UserID", string(server.IP+strconv.Itoa(server.Port)), "-sessionName", string(server.IP+strconv.Itoa(server.Port)), "-logFile", logFilePath,
		"-port", strconv.Itoa(port), "-region eu",
		"-serverName", server.IP, "-scene", server.MapName)

	err := cmd.Start()
	if err != nil {
		guiUpdate.AddLogMessage(fmt.Sprintf("Failed to start server %d: %v\n", server.ID, err))
		return
	}

	ServersMutex.Lock()
	for i, s := range Servers {
		if s.ID == server.ID {
			Servers[i].PID = cmd.Process.Pid
			guiUpdate.AddLogMessage(fmt.Sprintf("PID: %d Game server %d started on port %d. App server version %s .Map settings - %s .\n", Servers[i].PID, Servers[i].ID, Servers[i].Port, Servers[i].AppVersion, Servers[i].MapName))
			guiUpdate.RefreshServerList()
			break
		}
	}
	ServersMutex.Unlock()

	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Server %d stopped with error: %v\n", server.ID, err)
		}
	}()
}

func FindFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
