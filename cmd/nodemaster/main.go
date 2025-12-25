package main

import (
	"chain/internal/nodemaster"
	"context"
	"fmt"
)

func main() {
	AppMain(context.Background())
}

func AppMain(ctx context.Context) {
	err := nodemaster.StartServer()
	if err != nil {
		fmt.Println("start NodeMaster server error:", err)
	}
}
