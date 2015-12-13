package main

import (
	"flag"
	"strconv"
	"time"

	log "github.com/cihub/seelog"
	"github.com/stampzilla/stampzilla-go/nodes/basenode"
	"github.com/stampzilla/stampzilla-go/pkg/notifier"
	"github.com/stampzilla/stampzilla-go/protocol"
)

// MAIN - This is run when the init function is done

var notify *notifier.Notify

func main() { /*{{{*/
	log.Info("Starting SIMPLE node")

	// Parse all commandline arguments, host and port parameters are added in the basenode init function
	flag.Parse()

	//Get a config with the correct parameters
	config := basenode.NewConfig()

	//Activate the config
	basenode.SetConfig(config)

	node := protocol.NewNode("simple")

	//Start communication with the server
	connection := basenode.Connect()
	notify = notifier.New(connection)
	notify.SetSource(node)

	// Thit worker keeps track on our connection state, if we are connected or not
	go monitorState(node, connection)

	// Describe available actions
	//node.AddAction("set", "Set", []string{"Devices.Id"})
	//node.AddAction("toggle", "Toggle", []string{"Devices.Id"})
	//node.AddAction("dim", "Dim", []string{"Devices.Id", "value"})

	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeText,
		Name: "Example text",
		Command: &protocol.Command{
			Cmd:  "set",
			Args: []string{"1"},
		},
		Feedback: "Devices[0].State",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeButton,
		Name: "Example button",
		Command: &protocol.Command{
			Cmd:  "on",
			Args: []string{"1"},
		},
		Feedback: "Devices[1].State",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeToggle,
		Name: "Example toggle1",
		Command: &protocol.Command{
			Cmd:  "toggle",
			Args: []string{"1"},
		},
		Feedback: "Devices[1].State",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeToggle,
		Name: "Example toggle2",
		Command: &protocol.Command{
			Cmd:  "toggle",
			Args: []string{"2"},
		},
		Feedback: "Devices[2].State",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeSlider,
		Name: "Example slider",
		Command: &protocol.Command{
			Cmd:  "slider",
			Args: []string{"3"},
		},
		Feedback: "Devices[3].State",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeColorPicker,
		Name: "Example color picker",
		Command: &protocol.Command{
			Cmd:  "color",
			Args: []string{"1"},
		},
		Feedback: "Devices[4].State",
	})

	// Notification buttons
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeButton,
		Name: "Send \"Critical\" notification",
		Command: &protocol.Command{
			Cmd:  "notification",
			Args: []string{"Critical"},
		},
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeButton,
		Name: "Send \"Error\" notification",
		Command: &protocol.Command{
			Cmd:  "notification",
			Args: []string{"Error"},
		},
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeButton,
		Name: "Send \"Warning\" notification",
		Command: &protocol.Command{
			Cmd:  "notification",
			Args: []string{"Warning"},
		},
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeButton,
		Name: "Send \"Information\" notification",
		Command: &protocol.Command{
			Cmd:  "notification",
			Args: []string{"Information"},
		},
	})

	state := NewState()
	node.SetState(state)

	state.AddDevice("1", "Dev1", false)
	state.AddDevice("2", "Dev2", true)
	state.AddDevice("3", "Slider", 33)

	go startToggler(node, connection, "1", time.Second)

	// This worker recives all incomming commands
	go serverRecv(node, connection)
	select {}
} /*}}}*/

// WORKER that monitors the current connection state
func monitorState(node *protocol.Node, connection basenode.Connection) {
	for s := range connection.State() {
		switch s {
		case basenode.ConnectionStateConnected:
			connection.Send(node.Node())
		case basenode.ConnectionStateDisconnected:
		}
	}
}

// WORKER that recives all incomming commands
func serverRecv(node *protocol.Node, connection basenode.Connection) {
	for d := range connection.Receive() {
		processCommand(node, connection, d)
	}
}

// THis is called on each incomming command
func processCommand(node *protocol.Node, connection basenode.Connection, cmd protocol.Command) {
	if s, ok := node.State().(*State); ok {
		log.Infof("Incoming command from server: %#v \n", cmd)
		if len(cmd.Args) == 0 {
			return
		}
		device := s.Device(cmd.Args[0])

		switch cmd.Cmd {
		case "notification":
			switch cmd.Args[0] {
			case "Critical":
				notify.Critical("Test notifcation with level '" + cmd.Args[0] + "'")
			case "Error":
				notify.Error("Test notifcation with level '" + cmd.Args[0] + "'")
			case "Warning":
				notify.Warn("Test notifcation with level '" + cmd.Args[0] + "'")
			case "Information":
				notify.Info("Test notifcation with level '" + cmd.Args[0] + "'")
			}
		case "on":
			log.Info("got on")
			device.SetState(true)
			connection.Send(node.Node())
		case "off":
			log.Info("got off")
			device.SetState(false)
			connection.Send(node.Node())
		case "slider":
			i, err := strconv.Atoi(cmd.Params[0])
			if err == nil {
				device.SetState(i)
				connection.Send(node.Node())
			}
		case "toggle":
			log.Info("got toggle")
			if v, ok := device.State.(bool); ok {
				if v {
					device.SetState(false)
				} else {
					device.SetState(true)
				}
			}
			connection.Send(node.Node())
		}
	}
}

func startToggler(node *protocol.Node, connection basenode.Connection, deviceName string, interval time.Duration) {
	for {
		<-time.After(interval)
		if s, ok := node.State().(*State); ok {
			device := s.Device(deviceName)
			if v, ok := device.State.(bool); ok {
				if v {
					device.SetState(false)
				} else {
					device.SetState(true)
				}
			}

			connection.Send(node.Node())
		}
	}
}
