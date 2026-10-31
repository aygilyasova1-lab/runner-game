package game

import "os"

// serverAddr - адрес сервера рейтинга по умолчанию. Его можно вшить в exe при сборке:
//
//	go build -ldflags "-X game.go/game.serverAddr=1.2.3.4:50051" -o game.exe .
var serverAddr = "localhost:50051"

// serverAddress: переменная окружения GAME_SERVER важнее вшитого адреса.
func serverAddress() string {
	if v := os.Getenv("GAME_SERVER"); v != "" {
		return v
	}
	return serverAddr
}
