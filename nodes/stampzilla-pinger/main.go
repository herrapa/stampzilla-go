package main

import (
	"flag"

	log "github.com/cihub/seelog"
	"github.com/stampzilla/stampzilla-go/nodes/basenode"
	"github.com/stampzilla/stampzilla-go/protocol"
)

var VERSION string = "dev"
var BUILD_DATE string = ""

// GLOBAL VARS
var node *protocol.Node

type Config struct {
	Targets []ConfigTarget `json:"targets"`
}

type ConfigTarget struct {
	Name    string `json:"name"`
	Address string `json:"ip"`
}

// MAIN - This is run when the init function is done
func main() {
	log.Info("Starting PINGER node")
	// Create new node description
	// Parse all commandline arguments, host and port parameters are added in the basenode init function
	flag.Parse()

	//Get a config with the correct parameters
	config := basenode.NewConfig()

	//Activate the config
	basenode.SetConfig(config)

	nc := &Config{}
	err := config.NodeSpecific(&nc)
	if err != nil {
		log.Error(err)
		return
	}

	node = protocol.NewNode("pinger")
	node.Version = VERSION
	node.BuildDate = BUILD_DATE

	//Start communication with the server
	connection := basenode.Connect()

	// Thit worker keeps track on our connection state, if we are connected or not
	go monitorState(connection)

	// This worker recives all incomming commands
	go serverRecv(connection.Receive())

	state = targetState{
		Targets:    make(map[string]*Target),
		connection: connection,
	}
	node.SetState(state)

	for _, v := range nc.Targets {
		t := &Target{
			Name: v.Name,
			Ip:   v.Address,
		}
		state.add(t)
	}

	select {}
}

// WORKER that monitors the current connection state
func monitorState(connection basenode.Connection) {
	for s := range connection.State() {
		switch s {
		case basenode.ConnectionStateConnected:
			connection.Send(node.Node())
		case basenode.ConnectionStateDisconnected:
		}
	}
}

// WORKER that recives all incomming commands
func serverRecv(recv chan protocol.Command) {
	for d := range recv {
		processCommand(d)
	}
}

// This is called on each incomming command
func processCommand(cmd protocol.Command) {
	log.Info("Incoming command from server:", cmd)
}
