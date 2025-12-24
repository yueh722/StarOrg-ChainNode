package main

import (
	"chain/internal/nodemaster"
	"fmt"
)

func main() {
	err := nodemaster.StartServer()
	if err != nil {
		fmt.Println("start NodeMaster server error:", err)
	}
}
