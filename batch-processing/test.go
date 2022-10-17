package main

type Config struct {
	name string
}

func test() {
	var c Config
	c.name = "test"

	c{name: "some"}
}
