package server_launcher

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/config"
	"github.com/Tagakama/ServerManager/internal/matchmaking/room"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Launcher interface {
	LaunchGameServer(settings *room.Room) bool
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

func (s *ServerLauncher) LaunchGameServer(settings *room.Room) bool {
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

	// Запускаем процесс
	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start server %d: %v\n", settings.ID, err)
		return false
	}

	// Создаем каналы для синхронизации
	serverStarted := make(chan bool, 1)
	serverFailed := make(chan error, 1)

	go func() {
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				serverFailed <- fmt.Errorf("timeout waiting for server start in log file")
				return
			case <-ticker.C:
				// Читаем и анализируем лог-файл
				content, err := os.ReadFile(logFilePath)
				if err != nil {
					// Файл может еще не существовать, это нормально
					continue
				}

				logContent := string(content)

				// Ищем признаки успешного запуска сервера
				if strings.Contains(logContent, "started on") {
					fmt.Printf("Server %d successfully started (found in log)\n", settings.ID)
					serverStarted <- true
					return
				}

			}
		}
	}()

	// Горутина для ожидания завершения процесса
	go func() {
		err := cmd.Wait()
		if err != nil {
			serverFailed <- fmt.Errorf("server process exited with error: %v \n", err)
		}
	}()

	// Ожидаем либо успешного запуска, либо ошибки, либо таймаут
	select {
	case <-serverStarted:
		fmt.Printf("Server %s started successfully, continuing...\n", unicName)
		return true

	case err := <-serverFailed:
		fmt.Printf("server failed to start: %v \n", err)
		return false

	case <-time.After(30 * time.Second): // Таймаут 30 секунд
		fmt.Printf("server startup timed out after 30 seconds\n")
		return false
	}

	// Теперь метод будет ждать, пока сервер не запустится
	// Дальнейший код выполнится только после успешного запуска сервера
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
