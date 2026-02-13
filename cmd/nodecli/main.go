package main

import (
	"chain/internal/nodecli"
	"fmt"
	"os"
)

//TIP <p>To run your code, right-click the code and select <b>RunApp</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>RunApp</b> menu item from here.</p>

func main() {

	rootCmd := nodecli.RootCmd
	rootCmd.AddCommand(
		nodecli.CreateInitNodeFileCommand())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
