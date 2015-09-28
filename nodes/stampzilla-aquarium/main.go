package main

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"syscall"
	"unsafe"

	"github.com/tarm/goserial"

	log "github.com/cihub/seelog"
	"github.com/stampzilla/stampzilla-go/nodes/basenode"
	"github.com/stampzilla/stampzilla-go/protocol"
)

type SerialConnection struct {
	Name string
	Baud int
	Port io.ReadWriteCloser
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

var node *protocol.Node
var state *State = &State{}
var serverConnection basenode.Connection
var ard *SerialConnection
var disp *SerialConnection
var frameCount int
var lastTemp float64
var phFilter []float64

func main() {
	config := basenode.NewConfig()
	basenode.SetConfig(config)

	var arddev string
	var dispdev string
	flag.StringVar(&arddev, "arduino-dev", "/dev/arduino", "Arduino serial port")
	flag.StringVar(&dispdev, "display-dev", "/dev/poleDisplay", "Display serial port")

	flag.Parse()

	phFilter = make([]float64, 0)

	log.Info("Starting Aquarium node")

	// Create new node description
	node = protocol.NewNode("aquarium")
	node.SetState(state)

	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "Water temperature",
		Feedback: `WaterTemperature`,
	})
	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "Water level - OK",
		Feedback: `WaterLevelOk`,
	})
	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "Filter - OK",
		Feedback: `FilterOk`,
	})

	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "Cooling",
		Feedback: `Cooling`,
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeToggle,
		Name: "Heating",
		Command: &protocol.Command{
			Cmd:  "Heating",
			Args: []string{},
		},
		Feedback: `Heating`,
	})

	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeToggle,
		Name: "Skimmer",
		Command: &protocol.Command{
			Cmd:  "Skimmer",
			Args: []string{},
		},
		Feedback: `Skimmer`,
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeToggle,
		Name: "Circulation pumps",
		Command: &protocol.Command{
			Cmd:  "CirculationPumps",
			Args: []string{},
		},
		Feedback: `CirculationPumps`,
	})

	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeSlider,
		Name: "White",
		Command: &protocol.Command{
			Cmd:  "dim",
			Args: []string{"white"},
		},
		Feedback: "Lights.White",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeSlider,
		Name: "Red",
		Command: &protocol.Command{
			Cmd:  "dim",
			Args: []string{"red"},
		},
		Feedback: "Lights.Red",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeSlider,
		Name: "Green",
		Command: &protocol.Command{
			Cmd:  "dim",
			Args: []string{"green"},
		},
		Feedback: "Lights.Green",
	})
	node.AddElement(&protocol.Element{
		Type: protocol.ElementTypeSlider,
		Name: "Blue",
		Command: &protocol.Command{
			Cmd:  "dim",
			Args: []string{"blue"},
		},
		Feedback: "Lights.Blue",
	})
	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "Cover temperature",
		Feedback: `Lights.Temperature`,
	})

	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     "pH probe",
		Feedback: `PH`,
	})

	serverConnection = basenode.Connect()
	go monitorState(serverConnection)

	// This worker recives all incomming commands
	go serverRecv(serverConnection)

	// Setup the serial connection
	ard = &SerialConnection{Name: arddev, Baud: 115200}
	ard.run(serverConnection, func(data string, connection basenode.Connection) {
		processArduinoData(data, connection)
	})

	disp = &SerialConnection{Name: dispdev, Baud: 9600}
	disp.run(serverConnection, func(data string, connection basenode.Connection) {
		log.Debug("Incomming from display", data)
	})

	go updateDisplay(serverConnection, disp)

	/*go func() {
		for {
			fmt.Print("\r", frameCount)
			<-time.After(time.Millisecond * 10)
		}
	}()*/

	select {}
}

var updateInhibit bool = false
var changed bool = false

func processArduinoData(msg string, connection basenode.Connection) { // {{{
	//log.Debug(msg)

	var prevState State = *state

	values := strings.Split(msg, "|")
	if len(values) != 10 {
		printTerminalStatus("Invalid length")
		//log.Warn("Invalid message: ", msg)
		return
	}

	// Temperature// {{{
	value, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return
	}
	if value != state.WaterTemperature {
		changed = true
	}
	state.WaterTemperature = value // }}}
	value, err = strconv.ParseFloat(values[4], 64)
	if err != nil {
		return
	}
	lastTemp = value

	// Filling stat0e // {{{
	value, err = strconv.ParseFloat(values[1], 64)
	if err != nil {
		return
	}
	if state.FillingTime != value {
		changed = true
	}
	state.FillingTime = value // }}}

	// Cooling %// {{{
	value, err = strconv.ParseFloat(values[2], 64)
	if err != nil {
		return
	}
	if state.Cooling != value {
		changed = true
	}
	state.Cooling = value // }}}

	bits := values[3][0]
	state.CirculationPumps = bits&0x01 != 0
	state.Skimmer = bits&0x02 != 0
	state.Heating = bits&0x04 != 0
	state.Filling = bits&0x08 == 0
	state.WaterLevelOk = bits&0x10 == 0
	state.FilterOk = bits&0x20 == 0

	light := strings.Split(values[6], "*")
	value, err = strconv.ParseFloat(light[0], 64)
	if err == nil {
		state.Lights.Red = value
	}
	value, err = strconv.ParseFloat(light[1], 64)
	if err == nil {
		state.Lights.Green = value
	}
	value, err = strconv.ParseFloat(light[2], 64)
	if err == nil {
		state.Lights.Blue = value
	}
	value, err = strconv.ParseFloat(light[3], 64)
	if err == nil {
		state.Lights.White = value
	}
	value, err = strconv.ParseFloat(values[7], 64)
	if err == nil {
		if value > 0 {
			state.Lights.Temperature = value
		}
	} else {
		state.Lights.Temperature = -1
	}

	pH := strings.Split(values[8], ":")
	value, err = strconv.ParseFloat(pH[0], 64)
	if err == nil {
		ph := ((673-value)/828*14+7)*1.5650273224 - 3.84
		if len(phFilter) < 200 {
			phFilter = append(phFilter, ph)
		} else {
			phFilter = append(phFilter[1:], ph)
			state.PH = Average(phFilter)
		}
	}

	air := strings.Split(values[9], ",")
	if len(air) == 2 {
		value, err = strconv.ParseFloat(air[0], 64)
		if err == nil && value <= 100 {
			state.Humidity = value
		}
		value, err = strconv.ParseFloat(air[1], 64)
		if err == nil && value > 0 && value < 100 {
			state.AirTemperature = value
		}
	}

	// Check if something have changed
	if !reflect.DeepEqual(prevState, *state) {
		changed = true
	}

	frameCount++
	//fmt.Print(frameCount, " - ", msg, " - ", bits, "\r")
	printTerminalStatus(msg)

	if !updateInhibit && changed {
		changed = false
		//fmt.Print("\n")

		connection.Send(node.Node())

		updateInhibit = true
		//log.Warn("Invalid pacakge: ", msg)
		go func() {
			<-time.After(time.Millisecond * 200)
			updateInhibit = false
		}()
	}
} // }}}

func Average(xs []float64) float64 {
	total := float64(0)
	for _, x := range xs {
		total += x
	}
	return total / float64(len(xs))
}
func updateDisplay(connection basenode.Connection, serial *SerialConnection) { // {{{
	for {
		select {
		case <-time.After(time.Second / 15):
			if serial.Port == nil {
				continue
			}

			var msg, row1, row2 string
			var wLevel string = "OK"

			if lastTemp == 0 {
				wLevel = "NO TEMP"
			}

			//if !state.WaterLevelOk {
			//wLevel = "WLVL"
			//}
			if state.FillingTime > 0 {
				wLevel = "FILL ERR"
				if state.FillingTime < 20000 {
					wLevel = "FILLING"
				}
			}

			//msg += "\x1B" // Reset
			//msg += "\x0E" // Clear
			//msg += "\x13" // Cursor on
			msg += "\x11" // Character over write mode
			//msg += "\x12" // Vertical scroll mode
			//msg += "\x13" // Cursor on
			msg += "\x14" // Cursor off

			msg += "\r"

			row1 = time.Now().Format("15:04:05") + strings.Repeat(" ", 12-len(wLevel)) + wLevel
			row2 = "Temp " + strconv.FormatFloat(state.WaterTemperature, 'f', 2, 64) + " Cool " + strconv.FormatFloat(state.Cooling, 'f', 2, 64)

			if 20-len(row1) > 0 {
				row1 += strings.Repeat(" ", 20-len(row1))
			}

			if 20-len(row2) > 0 {
				row2 += strings.Repeat(" ", 20-len(row2))
			}
			msg += row1[0:20]
			msg += row2[0:20]

			_, err := serial.Port.Write([]byte(msg))
			if err != nil {
				log.Debug("Failed write to display:", err)
			}
		}
	}
} // }}}

// WORKER that monitors the current connection state
func monitorState(connection basenode.Connection) { // {{{
	for s := range connection.State() {
		switch s {
		case basenode.ConnectionStateConnected:
			connection.Send(node.Node())
		case basenode.ConnectionStateDisconnected:
		}
	}
} // }}}

// WORKER that recives all incomming commands
func serverRecv(connection basenode.Connection) { // {{{
	for d := range connection.Receive() {
		if err := processCommand(d); err != nil {
			log.Error(err)
		}
	}
} // }}}

func processCommand(cmd protocol.Command) error { // {{{
	var target bool

	if len(cmd.Args) < 1 {
		if len(cmd.Params) < 1 {
			return fmt.Errorf("Missing arguments, ignoring command")
		} else {
			target = cmd.Params[0] != "" && cmd.Params[0] != "false"
		}
	} else {
		target = cmd.Args[0] != "" && cmd.Args[0] != "0"
	}

	switch cmd.Cmd {
	case "CirculationPumps":
		if target {
			ard.Port.Write([]byte{0x02, 0x01, 0x01, 0x03}) // Turn on
		} else {
			ard.Port.Write([]byte{0x02, 0x01, 0x00, 0x03}) // Turn off
		}
		break
	case "Skimmer":
		if target {
			ard.Port.Write([]byte{0x02, 0x02, 0x01, 0x03}) // Turn on
		} else {
			ard.Port.Write([]byte{0x02, 0x02, 0x00, 0x03}) // Turn off
		}
		break
	case "Heating":
		if target {
			ard.Port.Write([]byte{0x02, 0x03, 0x01, 0x03}) // Turn on
		} else {
			ard.Port.Write([]byte{0x02, 0x03, 0x00, 0x03}) // Turn off
		}
		break
	case "CoolingP":
		i, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("Failed to decode arg[0] to int %s %s", err, cmd.Args[0])
		}

		ard.Port.Write([]byte{0x02, 0x04, byte(i), 0x03}) // Turn on
		break
	case "CoolingI":
		i, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("Failed to decode arg[0] to int %s %s", err, cmd.Args[0])
		}

		ard.Port.Write([]byte{0x02, 0x05, byte(i), 0x03}) // Turn on
		break
	case "CoolingD":
		i, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("Failed to decode arg[0] to int %s %s", err, cmd.Args[0])
		}

		ard.Port.Write([]byte{0x02, 0x06, byte(i), 0x03}) // Turn on
		break
	case "dim":
		var i int
		var err error

		switch {
		case len(cmd.Args) == 1:
			i, err = strconv.Atoi(cmd.Params[0])
			if err != nil {
				return fmt.Errorf("Failed to decode param[0] to int %s %s", err, cmd.Args[0])
			}
		case len(cmd.Args) == 2:
			i, err = strconv.Atoi(cmd.Args[1])
			if err != nil {
				return fmt.Errorf("Failed to decode arg[1] to int %s %s", err, cmd.Args[0])
			}

		}

		switch cmd.Args[0] {
		case "red":
			ard.Port.Write([]byte{0x02, 0x07, byte(i), 0x03}) // Turn on
		case "green":
			ard.Port.Write([]byte{0x02, 0x08, byte(i), 0x03}) // Turn on
		case "blue":
			ard.Port.Write([]byte{0x02, 0x09, byte(i), 0x03}) // Turn on
		case "white":
			ard.Port.Write([]byte{0x02, 0x0A, byte(i), 0x03}) // Turn on
		}
	}
	return nil
} // }}}

func printTerminalStatus(msg string) {
	size := getWindowSize()

	if size.Col < 20 {
		return
	}

	if len(msg) > int(size.Col)-1 {
		msg = msg[:size.Col]
	}

	log.Flush()

	// Header
	fmt.Println(strings.Repeat(" ", int(size.Col)-1))
	fmt.Println(strings.Repeat("-", int(size.Col)-1))

	// Raw message
	printWithLimit(msg, int(size.Col))

	// Framecount
	msg = "Framecount: " + strconv.Itoa(frameCount)
	printWithLimit(msg, int(size.Col))

	fmt.Print("\033[4A\r") // Move cursor up 2 lines
}

func printWithLimit(msg string, length int) {
	pad := length - len(msg) - 1

	if pad < 1 {
		fmt.Print(msg[:len(msg)+pad])
	} else {
		fmt.Print(msg + strings.Repeat(" ", pad) + "\n")
	}
}

func getWindowSize() *winsize {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return ws
}

// Serial connection workers
func (config *SerialConnection) run(connection basenode.Connection, callback func(data string, connection basenode.Connection)) { // {{{
	go func() {
		for {
			config.connect(connection, callback)
			<-time.After(time.Second * 4)
		}
	}()
}                                                                                                                                     // }}}
func (config *SerialConnection) connect(connection basenode.Connection, callback func(data string, connection basenode.Connection)) { // {{{

	red := state.Lights.Red
	green := state.Lights.Green
	blue := state.Lights.Blue
	white := state.Lights.White

	c := &serial.Config{Name: config.Name, Baud: config.Baud}
	var err error

	config.Port, err = serial.OpenPort(c)
	if err != nil {
		log.Error("Serial connect failed: ", err)
		return
	}

	<-time.After(time.Second)

	config.Port.Write([]byte{0x02, 0x07, byte(red), 0x03})   // red
	config.Port.Write([]byte{0x02, 0x08, byte(green), 0x03}) // green
	config.Port.Write([]byte{0x02, 0x09, byte(blue), 0x03})  // blue
	config.Port.Write([]byte{0x02, 0x0A, byte(white), 0x03}) // white

	var incomming string = ""

	for {
		buf := make([]byte, 128)

		n, err := config.Port.Read(buf)
		if err != nil {
			log.Error("Serial read failed: ", err)
			return
		}

		incomming += string(buf[:n])

		for {
			n := strings.Index(incomming, "\r")

			if n < 0 {
				break
			}

			msg := strings.TrimSpace(incomming[:n])
			incomming = incomming[n+1:]

			go callback(msg, connection)
		}
	}
} // }}}
