package handlers_test

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	handlers "github.com/Tagakama/ServerManager/internal/tcp-server/handlers/tcp/handleConnection"
	"github.com/Tagakama/ServerManager/internal/tcp-server/type"
)

// Упростим мок
type MockWorkerPool struct {
	AddedTasks []_type.PendingConnection
}

func (m *MockWorkerPool) AddTask(task _type.PendingConnection) {
	m.AddedTasks = append(m.AddedTasks, task)
}

// Генератор случайной строки
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Тест
func TestHandleConnection_MultipleRandomRequests(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		shouldAdd    bool
		expectClient string
	}{
		{
			name:         "Valid message 1",
			input:        "client1:Join:3:Forest:v1.0.0\n",
			shouldAdd:    true,
			expectClient: "client1",
		},
		{
			name: "Valid random",
			input: fmt.Sprintf("%s:Play:%d:%s:v%s\n",
				randomString(8),
				rand.Intn(10),
				randomString(5),
				fmt.Sprintf("%d.%d.%d", rand.Intn(2), rand.Intn(10), rand.Intn(10)),
			),
			shouldAdd: true,
		},
		{
			name:      "Invalid message (missing fields)",
			input:     "brokenmessage\n",
			shouldAdd: false,
		},
		{
			name:      "Invalid int conversion",
			input:     "user:Run:NaN:Map:v1.0\n",
			shouldAdd: true, // добавится, но NumberOfPlayers будет 0
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Case_%d_%s", i, tc.name), func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			mockPool := &MockWorkerPool{}

			// Пишем сообщение
			go func() {
				client.Write([]byte(tc.input))
			}()

			handlers.HandleConnection(server, mockPool)

			if tc.shouldAdd && len(mockPool.AddedTasks) != 1 {
				t.Fatalf("Expected task to be added")
			}
			if !tc.shouldAdd && len(mockPool.AddedTasks) != 0 {
				t.Fatalf("Expected no task to be added")
			}

			// Дополнительные проверки
			if tc.shouldAdd && tc.expectClient != "" {
				if mockPool.AddedTasks[0].ConnectedMessage.ClientID != tc.expectClient {
					t.Errorf("Expected ClientID to be %s, got %s",
						tc.expectClient, mockPool.AddedTasks[0].ConnectedMessage.ClientID)
				}
			}
		})
	}
}

func TestHandleConnection_StressTest100000(t *testing.T) {
	const numMessages = 100000

	mockPool := &MockWorkerPool{}

	for i := 0; i < numMessages; i++ {
		server, client := net.Pipe()

		// Генерация валидного сообщения
		clientID := fmt.Sprintf("client%d", i)
		action := randomString(6)
		numPlayers := rand.Intn(10) + 1
		mapName := randomString(5)
		version := fmt.Sprintf("v%d.%d.%d", rand.Intn(3), rand.Intn(10), rand.Intn(10))

		msg := fmt.Sprintf("%s:%s:%d:%s:%s\n", clientID, action, numPlayers, mapName, version)

		// Пишем в соединение
		go func(message string, c net.Conn) {
			_, err := c.Write([]byte(message))
			if err != nil {
				t.Errorf("Error writing to connection: %v", err)
			}
			c.Close()
		}(msg, client)

		// Обработка соединения
		handlers.HandleConnection(server, mockPool)
		server.Close()
	}

	// Проверка
	if len(mockPool.AddedTasks) != numMessages {
		t.Fatalf("Expected %d tasks to be added, got %d", numMessages, len(mockPool.AddedTasks))
	}

	// Поверхностная проверка содержимого
	for i, task := range mockPool.AddedTasks {
		if task.ConnectedMessage.ClientID == "" {
			t.Errorf("Task %d has empty ClientID", i)
		}
		if task.ConnectedMessage.NumberOfPlayers <= 0 {
			t.Errorf("Task %d has invalid NumberOfPlayers: %d", i, task.ConnectedMessage.NumberOfPlayers)
		}
	}
}
