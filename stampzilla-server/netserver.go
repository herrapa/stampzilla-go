package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/cihub/seelog"
)

func netStart(port string) {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("listen error", err)
		return
	}

	go func() {
		for {
			fd, err := l.Accept()
			if err != nil {
				fmt.Println("accept error", err)
				return
			}

			go newClient(fd)
		}
	}()
}

func newClient(connection net.Conn) {
	// Recive data
	log.Info("New client connected")
	id := ""
	uuid := ""
	for {
		reader := bufio.NewReader(connection)
		decoder := json.NewDecoder(reader)
		var info Node
		err := decoder.Decode(&info)

		//err = json.Unmarshal(data, &cmd)
		if err != nil {
			if err.Error() == "EOF" {
				log.Info(id, " - Client disconnected")
				if uuid != "" {
					nodes.Delete(uuid)
				}
				//TODO be able to not send everything always.
				clients.messageOtherClients(&Message{"all", nodes})
				return
			}
			log.Warn("Not disconnect but error: ", err)
			//return here?
		} else {
			id = info.Id
			uuid = info.Uuid
			info.conn = connection

			nodes.Add(&info)
			log.Info(info.Id, " - Got update on state")
			clients.messageOtherClients(&Message{"singlenode", info})
		}

	}

}

// Handle a client
//func newClientOld(c net.Conn) {
//log.Info("New client connected")
//id := ""
//for {
//buf := make([]byte, 51200)
//nr, err := c.Read(buf)
//if err != nil {
//log.Info(id, " - Client disconnected")
//if id != "" {
//delete(nodes, id)
//}
////TODO be able to not send everything always.
//clients.messageOtherClients(&Message{"all", nodes})
//return
//}

////TODO: Handle when multiple messages gets concated ex: msg}{msg2
////  TODO: see possible solution above in the new improved newClient function :)  (jonaz) <Fri 10 Oct 2014 10:04:43 AM CEST>

//data := buf[0:nr]

//var info Node
//err = json.Unmarshal(data, &info)
//if err != nil {
//log.Warn(err, " -->", string(data), "<--")
//} else {
//id = info.Id
//nodes[info.Id] = info
//nodesConnection[info.Id] = &NodeConnection{conn: c, wait: nil}

//log.Info(info.Id, " - Got update on state")

//if nodesConnection[info.Id].wait != nil {
//select {
//case nodesConnection[info.Id].wait <- false:
//close(nodesConnection[info.Id].wait)
//nodesConnection[info.Id].wait = nil
//default:
//}
//}

//clients.messageOtherClients(&Message{"singlenode", nodes[info.Id]})
//}
//}
//}