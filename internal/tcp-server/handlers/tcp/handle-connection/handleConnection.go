package handlers

import (
	"bufio"
	"fmt"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
	"net"
	"strconv"
	"strings"
	"time"
)

func HandleConnection(conn net.Conn, pool workers.TaskSubmitter) {

	reader := bufio.NewReader(conn)
	rawMessage, err := reader.ReadString('\n')
	rawMessage = strings.TrimSpace(rawMessage)
	if err != nil {
		fmt.Println("Error reading from connection: ", err)
	}

	var clientConnection = func() (*_type.PendingConnection, error) {
		handleRawMessage := strings.SplitN(rawMessage, ":", -1)
		//newMessage := _type.Message{}
		//if len(handleRawMessage) != reflect.TypeOf(newMessage).NumField() {
		//	fmt.Sprintf("Error message format :%s", rawMessage)
		//	return &_type.PendingConnection{Conn: conn}, fmt.Errorf("Format not allowed")
		//}
		return &_type.PendingConnection{Conn: conn,
			ConnectedMessage: _type.Message{ClientID: handleRawMessage[0],
				Message: handleRawMessage[1],
				NumberOfPlayers: func(s string) int {
					i, err := strconv.Atoi(s)
					if err != nil {
						fmt.Println("Error converting NumberOfPlayers to int: ", err)
						return 0
					}

					if i <= 0 {
						return 1
					}

					return i
				}(handleRawMessage[2]),
				MapName:    handleRawMessage[3],
				AppVersion: handleRawMessage[4],
			}}, nil
	}

	pendingConnection, err := clientConnection()
	if err != nil {
		fmt.Println("Error creating pending connection: ", err)
		return
	}

	fmt.Printf("New request - Time: %s, Client: %s, Map: %s, Player count: %d\n",
		time.Now().Format("02-01-2006 15:04:05"),
		pendingConnection.ConnectedMessage.ClientID,
		pendingConnection.ConnectedMessage.MapName,
		pendingConnection.ConnectedMessage.NumberOfPlayers)

	pool.AddTask(pendingConnection)

}
