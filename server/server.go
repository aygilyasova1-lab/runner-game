package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	pb "game.go/gamepb" // замените YOUR_MODULE на имя модуля из go.mod
	"game.go/service"
	"game.go/storage"
)

func main() {
	store, err := storage.Open("game.db")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLeaderboardServer(grpcServer, service.New(store))

	log.Println("сервер запущен на :50051, база: game.db")
	log.Fatal(grpcServer.Serve(lis))
}
