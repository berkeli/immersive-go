package server

type LogEntry struct {
	Term    int
	Command interface{}
}
