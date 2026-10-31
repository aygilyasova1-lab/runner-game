package main

import (
	"log"

	"game.go/game" // замените YOUR_MODULE на имя модуля из go.mod
)

func main() {
	if err := game.Run(); err != nil {
		log.Fatal(err)
	}
}
