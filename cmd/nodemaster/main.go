package main

import (
	"chain/internal/nodemaster"
	"context"
	"log"
)

func main() {
	AppMain(context.Background())
}

func AppMain(ctx context.Context) {
	err := nodemaster.StartServer()
	if err != nil {
		log.Println("start NodeMaster server error:", err)
	}
}
