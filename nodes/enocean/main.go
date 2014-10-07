// Package main provides ...
package main

import (
	"flag"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/jonaz/goenocean"
	"github.com/stampzilla/stampzilla-go/nodes/basenode"
	"github.com/stampzilla/stampzilla-go/protocol"
)

var node *protocol.Node
var state *State
var serverSendChannel chan interface{}
var serverRecvChannel chan protocol.Command

func init() {
	var host string
	var port string
	flag.StringVar(&host, "host", "192.168.13.2", "Server host/ip")
	flag.StringVar(&port, "port", "8282", "Server port")

	flag.Parse()

	//Setup Config
	config := &basenode.Config{
		Host: host,
		Port: port}

	basenode.SetConfig(config)

	//Start communication with the server
	serverSendChannel = make(chan interface{})
	serverRecvChannel = make(chan protocol.Command)
	connectionState := basenode.Connect(serverSendChannel, serverRecvChannel)
	go monitorState(connectionState, serverSendChannel)
	go serverRecv(serverRecvChannel)

}

func main() {

	node = protocol.NewNode("enocean")
	// Describe available actions
	node.AddAction("set", "Set", []string{"Devices.Id"})
	node.AddAction("toggle", "Toggle", []string{"Devices.Id"})
	node.AddAction("dim", "Dim", []string{"Devices.Id", "value"})

	// Describe available layouts
	node.AddLayout("1", "switch", "toggle", "Devices", []string{"on"}, "Switches")
	node.AddLayout("2", "slider", "dim", "Devices", []string{"dim"}, "Dimmers")
	node.AddLayout("3", "slider", "dim", "Devices", []string{"dim"}, "Specials")

	state = NewState()
	//state.AddDevice([4]byte{1, 2, 3, 4}, "Testdevice", []string{"asdf"}, "off")
	node.SetState(state)

	setupEnoceanCommunication()
}

func monitorState(connectionState chan int, send chan interface{}) {
	for s := range connectionState {
		switch s {
		case basenode.ConnectionStateConnected:
			send <- node
		case basenode.ConnectionStateDisconnected:
		}
	}
}

func serverRecv(recv chan protocol.Command) {

	for d := range recv {
		processCommand(d)
	}

}

func processCommand(cmd protocol.Command) {
	log.Error("INCOMING COMMAND", cmd)
}

func setupEnoceanCommunication() {
	send := make(chan goenocean.Encoder)
	recv := make(chan goenocean.Packet)
	goenocean.Serial(send, recv)

	go testSend(send)
	reciever(recv)
}

func testSend(send chan goenocean.Encoder) {
	p := goenocean.NewTelegramRps()
	p.SetTelegramData([]byte{0x50}) //on
	//p.SetStatus(0x30) //testing shows this does not need to be set! Status defaults to 0

	fmt.Println("Sending:", p.Encode())
	send <- p

	time.Sleep(time.Second * 3)
	p.SetTelegramData([]byte{0x70}) //off
	send <- p
}

func reciever(recv chan goenocean.Packet) {

	for p := range recv {
		if p.SenderId() != [4]byte{0, 0, 0, 0} {
			incomingPacket(p)
		}
	}

}

func incomingPacket(p goenocean.Packet) {

	var d *Device
	if d = state.Device(p.SenderId()); d == nil {
		//Add unknown device
		d = state.AddDevice(p.SenderId(), "UNKNOWN", nil, "")
		if d.Id() == "0186ff7d" {
			d.AddEep("a51201")
			d.AddEep("d20109")
		}
		serverSendChannel <- node
	}

	if b, ok := p.(goenocean.Telegram); ok {
		switch b.TelegramType() {
		case goenocean.TelegramTypeVld:
			fmt.Println("VLD TELEGRAM DETECTED")
			incomingVldTelegram(d, b)
		case goenocean.TelegramType4bs:
			fmt.Println("4BS TELEGRAM DETECTED")
			incoming4bsTelegram(d, b)
		}
	}

}
func incomingVldTelegram(d *Device, t goenocean.TelegramVld) {
	for _, deviceEep := range d.EEPs {
		switch deviceEep {
		case "d20109":
			eep := goenocean.NewEepD20109()
			eep.SetTelegram(t) //THIS IS COOL!
			fmt.Println("OUTPUTVALUE", eep.OutputValue())
			if eep.CommandId() == 4 {
				if eep.OutputValue() > 0 {
					d.State = "ON"
					//d.State = "ON"
				} else {
					//d.State = "OFF"
					d.State = "OFF"
				}
			}
		}

	}

	serverSendChannel <- node
}

func incoming4bsTelegram(d *Device, t goenocean.Telegram4bs) {
	for _, deviceEep := range d.EEPs {
		switch deviceEep {
		case "a51201":
			eep := goenocean.NewEepA51201()
			eep.SetTelegram(t) //THIS IS COOL!
			d.SetPower(eep.MeterReading())
			d.PowerUnit = eep.DataType()
		}

	}

	serverSendChannel <- node
}
