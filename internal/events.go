package internal

type CommandStreamMsg struct {
	Command *Command
	Line    string
	Err     error
	Done    bool
}

type FileEventMsg struct {
	Command *Command
	Message string
}
