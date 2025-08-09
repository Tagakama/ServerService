package main_test

import (
	"context"
	"fmt"
	handlers "github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/handleConnection"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Tagakama/ServerManager/internal/config"
	server_launcher "github.com/Tagakama/ServerManager/internal/game-server/server-launcher"
	"github.com/Tagakama/ServerManager/internal/matchmaking/matchmaker"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
	"github.com/stretchr/testify/require"
)

const (
	testPort    = "8081"
	testAddress = "127.0.0.1"
)

func TestFullCycle(t *testing.T) {
	// 1. Подготовка тестовой конфигурации
	cfg := &config.Config{
		VersionPath:    "./test_versions/",
		ExecutableName: "test_game_server",
		TCPServer: config.TCPServer{
			Address:     testAddress,
			Port:        testPort,
			WorkerCount: 3,
		},
	}

	// 2. Инициализация компонентов
	sl := server_launcher.New(cfg)
	mm := matchmaker.NewMatchmaker(sl)
	wp := workers.NewWorkerPool(cfg.WorkerCount, mm)

	// 3. Запуск сервера с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverManagerListener, err := net.Listen("tcp", net.JoinHostPort(cfg.Address, cfg.Port))
		if err != nil {
			serverErr <- fmt.Errorf("failed to start server: %w", err)
			return
		}
		defer serverManagerListener.Close()

		go func() {
			<-ctx.Done()
			serverManagerListener.Close()
		}()

		for {
			conn, err := serverManagerListener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					// Ожидаемый случай при завершении теста
					return
				default:
					serverErr <- fmt.Errorf("error accepting connection: %w", err)
					return
				}
			}
			go handlers.HandleConnection(conn, wp)
		}
	}()

	// Даем время серверу запуститься
	select {
	case err := <-serverErr:
		t.Fatalf("Server failed to start: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	// 4. Тестовые сценарии
	t.Run("single client connection", func(t *testing.T) {
		conn, err := net.Dial("tcp", net.JoinHostPort(testAddress, testPort))
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write([]byte("client1:join:2:map1:v1.0\n"))
		require.NoError(t, err)

		// Читаем ответ с таймаутом
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil && !os.IsTimeout(err) {
			t.Errorf("Error reading response: %v", err)
		}
	})

	// 5. Проверяем, что сервер корректно обработал соединение
	require.Eventually(t, func() bool {
		return len(mm.CurrentRooms) > 0
	}, time.Second, 100*time.Millisecond, "No rooms created after client connection")

	// 6. Завершаем тест
	cancel()
}
