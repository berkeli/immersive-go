package main

import (
	"flag"
	"multiple-servers/static"
)

var (
	flagPath string
	flagPort int
)

func init() {
	defer flag.Parse()
	flag.StringVar(&flagPath, "path", "assets", "Provide the path to the static files")
	flag.IntVar(&flagPort, "port", 8080, "Provide the port where static server will listen")
}

func main() {
	static.Run(flagPath, flagPort)
}
