package main

import (
	"flag"

	log "github.com/cihub/seelog"
	"github.com/stampzilla/stampzilla-go/stampzilla-server/logic"
	"github.com/stampzilla/stampzilla-go/stampzilla-server/protocol"
)

var netPort string
var webPort string
var webRoot string

var nodes *protocol.Nodes
var logicHandler *logic.Logic

func main() {

	flag.StringVar(&netPort, "net-port", "8282", "Stampzilla server port")
	flag.StringVar(&webPort, "web-port", "8080", "Webserver port")
	flag.StringVar(&webRoot, "web-root", "public", "Webserver root")
	flag.Parse()

	clients = newClients()
	logicHandler = logic.NewLogic()

	// Load logger
	logger, err := log.LoggerFromConfigAsFile("logconfig.xml")
	if err != nil {
		testConfig := `
        <seelog type="sync">
            <outputs formatid="main">
                <console/>
            </outputs>
            <formats>
                <format id="main" format="%Ns [%Level] %Msg%n"/>
            </formats>
        </seelog>`

		logger, _ := log.LoggerFromConfigAsBytes([]byte(testConfig))
		log.ReplaceLogger(logger)
	} else {
		log.ReplaceLogger(logger)
	}

	nodes = protocol.NewNodes()

	log.Info("Starting NET (:" + netPort + ")")
	netStart(netPort)

	log.Info("Starting WEB (:" + webPort + " in " + webRoot + ")")
	webStart(webPort, webRoot)

}
