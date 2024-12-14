package main

import "bytes"

type CommandService struct {
	inputChan    chan Command
	RecentOutput map[string]string
}

func NewCommandService() *CommandService {
	return &CommandService{
		inputChan:    make(chan Command),
		RecentOutput: make(map[string]string),
	}
}

type ChannelWriter struct {
	channel chan string
}

func (cw *ChannelWriter) Write(p []byte) (n int, err error) {
	cw.channel <- string(p)
	return len(p), nil
}

func (cs *CommandService) Start() {
	go func() {
		for cmd := range cs.inputChan {
			// Simulate processing time
			// time.Sleep(1 * time.Second)
			cs.RecentOutput[cmd.Target] = "Running command"
			buffer := new(bytes.Buffer)
			err := ExecuteCommand(cmd.Shell, buffer)
			if err != nil {
				cs.RecentOutput[cmd.Target] = err.Error() + "\n\n" + buffer.String()
			} else {
				if cmd.StreamOutput {
					cs.RecentOutput[cmd.Target] = buffer.String()
				} else {
					cs.RecentOutput[cmd.Target] = "Command execution successful"
				}
				// cs.RecentOutput[cmd.Target] = "Command execution successful"
			}
			// cs.outputChan <- fmt.Sprintf("Processed: %s", cmd)

		}
	}()
}

func (cs *CommandService) SendCommand(cmd Command) {
	cs.inputChan <- cmd
}

func (cs *CommandService) GetOutput(target string) string {
	return cs.RecentOutput[target]
}
