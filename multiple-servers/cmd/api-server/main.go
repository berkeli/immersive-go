package main

import (
	"flag"
	"multiple-servers/api"
	"os"
)

var (
	flagPort int
	DB_URL   string
)

func init() {
	defer flag.Parse()
	flag.IntVar(&flagPort, "port", 8080, "Provide the port where static server will listen")
	DB_URL = os.Getenv("DATABASE_URL")
}

func main() {
	api.Run(DB_URL, flagPort)
}
