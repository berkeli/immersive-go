package client

import "fmt"

func PrintHelp() {
	fmt.Println("RAFT-OTEL CLI")
	fmt.Println("You can use the following commands:")
	fmt.Println("set <key> <value>")
	fmt.Println("get <key>")
	fmt.Println("setif <key> <value> <prev value>")
	fmt.Println("help to see this message again")
	fmt.Println("exit to exit the program")
	fmt.Println("---------------------")
}
