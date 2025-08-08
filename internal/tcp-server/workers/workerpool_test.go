package workers_test

import (
	_ "errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
)

type dummyConn struct{}

func (d dummyConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (d dummyConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (d dummyConn) Close() error                       { return nil }
func (d dummyConn) LocalAddr() net.Addr                { return nil }
func (d dummyConn) RemoteAddr() net.Addr               { return nil }
func (d dummyConn) SetDeadline(t time.Time) error      { return nil }
func (d dummyConn) SetReadDeadline(t time.Time) error  { return nil }
func (d dummyConn) SetWriteDeadline(t time.Time) error { return nil }

func makeFakeTask() _type.PendingConnection {
	return _type.PendingConnection{
		Conn: dummyConn{},
		ConnectedMessage: _type.Message{
			ClientID:        "test-client",
			Message:         "test-msg",
			NumberOfPlayers: 2,
			MapName:         "TestMap",
			AppVersion:      "1.0.0",
		},
	}
}

func TestNewWorkerPool_InvalidWorkerCount(t *testing.T) {
	pool, err := workers.NewWorkerPool(0)
	if err == nil {
		t.Error("Expected error for 0 workers, got nil")
	}
	if pool != nil {
		t.Error("Expected nil pool on error")
	}
}

func TestNewWorkerPool_ValidWorkerCount(t *testing.T) {
	pool, err := workers.NewWorkerPool(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	defer pool.Close()
}

func TestAddTask_Success(t *testing.T) {
	pool, _ := workers.NewWorkerPool(1)
	defer pool.Close()

	err := pool.AddTask(makeFakeTask())
	if err != nil {
		t.Errorf("Expected successful AddTask, got error: %v", err)
	}
}

func TestAddTask_AfterClose(t *testing.T) {
	pool, _ := workers.NewWorkerPool(1)
	_ = pool.Close()

	err := pool.AddTask(makeFakeTask())
	if err == nil || err.Error() != "Worker pool is closed" {
		t.Errorf("Expected 'Worker pool is closed', got: %v", err)
	}
}

func TestAddTask_PoolFull(t *testing.T) {
	pool, _ := workers.NewWorkerPool(1)
	defer pool.Close()

	// заполним канал
	for i := 0; i < 100; i++ {
		_ = pool.AddTask(makeFakeTask())
	}

	// 101-я задача должна не влезть
	err := pool.AddTask(makeFakeTask())
	if err == nil || err.Error() != "Worker pool is full" {
		t.Errorf("Expected 'Worker pool is full', got: %v", err)
	}
}

func TestClose_Twice(t *testing.T) {
	pool, _ := workers.NewWorkerPool(1)
	err := pool.Close()
	if err != nil {
		t.Fatalf("unexpected error on first close: %v", err)
	}

	err = pool.Close()
	if err == nil || err.Error() != "Worker pool is closed" {
		t.Errorf("Expected error on second close, got: %v", err)
	}
}

func TestWorkerPool_ProcessesTasks(t *testing.T) {
	pool, _ := workers.NewWorkerPool(2)
	defer pool.Close()

	for i := 0; i < 5; i++ {
		err := pool.AddTask(makeFakeTask())
		if err != nil {
			t.Fatalf("unexpected error adding task: %v", err)
		}
	}

	// Дать немного времени на выполнение (worker просто fmt.Sprintf пишет)
	time.Sleep(100 * time.Millisecond)
}

func TestWorkerPool_Stress(t *testing.T) {
	wp, _ := workers.NewWorkerPool(4)
	defer wp.Close()

	mockConn := dummyConn{}

	for i := 0; i < 1000; i++ {
		task := workers.Task{ID: i,
			Request: _type.PendingConnection{
				Conn: mockConn,
				ConnectedMessage: _type.Message{
					ClientID: fmt.Sprintf("Client-%d", i),
					Message:  fmt.Sprintf("Message-%d", i),
				},
			},
		}
		wp.Submit(task)
	}

}
