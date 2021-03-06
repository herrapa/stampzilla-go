package main

import (
	"io/ioutil"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/logic"
	serverprotocol "github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/protocol"
	"github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/servernode"
	"github.com/stampzilla/stampzilla-go/protocol"
)

type WebHandler struct {
	Logic          *logic.Logic            `inject:""`
	Scheduler      *logic.Scheduler        `inject:""`
	Nodes          *serverprotocol.Nodes   `inject:""`
	Devices        *serverprotocol.Devices `inject:""`
	NodeServer     *NodeServer             `inject:""`
	ActionsService *logic.ActionService    `inject:""`
}

func (wh *WebHandler) GetNodes(c *gin.Context) {
	c.JSON(200, wh.Nodes.All())
}

func (wh *WebHandler) GetNode(c *gin.Context) {
	if n := wh.Nodes.Search(c.Param("id")); n != nil {
		c.JSON(200, n)
		return
	}
	c.String(404, "{}")
}

func (wh *WebHandler) CommandToNodePut(c *gin.Context) {
	id := c.Param("id")
	requestJsonPut, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(err)
		c.String(500, "Error")
	}

	node := wh.Nodes.Search(id)
	if node == nil {
		log.Debug("NODE: ", node)
		c.String(404, "Node not found")
	}

	log.Infof("Sending to node %s command: %#v", id, string(requestJsonPut))

	serverprotocol.WriteUpdate(node, protocol.NewUpdateWithData(protocol.TypeCommand, requestJsonPut))
	c.JSON(200, protocol.Command{Cmd: "testresponse"})
}

func (wh *WebHandler) CommandToNodeGet(c *gin.Context) {
	id := c.Param("id")
	node := wh.Nodes.Search(id)
	if node == nil {
		c.String(404, "Node not found")
		return
	}

	// Split on / to add arguments
	p := strings.Split(c.Param("cmd"), "/")
	cmd := protocol.Command{Cmd: p[1], Args: p[2:]}

	log.Infof("Sending to node %s command: %#v", id, cmd)

	serverprotocol.WriteUpdate(node, protocol.NewUpdateWithData(protocol.TypeCommand, cmd))
	c.JSON(200, protocol.Command{Cmd: "testresponse"})
}

func (wh *WebHandler) GetActions(c *gin.Context) {
	if p := c.Request.URL.Query()["reload"]; len(p) > 0 {
		wh.ActionsService.Start()
	}
	c.JSON(200, wh.ActionsService.Get())
}
func (wh *WebHandler) ReloadActions(c *gin.Context) {
	wh.ActionsService.Start()
	c.JSON(200, wh.ActionsService.Get())
}

func (wh *WebHandler) RunAction(c *gin.Context) {
	uuid := c.Param("uuid")
	if a := wh.ActionsService.GetByUuid(uuid); a != nil {
		a.Run(wh.Logic.ActionProgressChan)
		c.JSON(200, "ok")
		return
	}
	c.String(404, "Action not found")
}

func (wh *WebHandler) GetAction(c *gin.Context) {
	uuid := c.Param("uuid")
	if a := wh.ActionsService.GetByUuid(uuid); a != nil {
		c.JSON(200, a)
		return
	}
	c.String(404, "Action not found")
}
func (wh *WebHandler) GetRules(c *gin.Context) {
	if p := c.Request.URL.Query()["reload"]; len(p) > 0 {
		wh.Logic.RestoreRulesFromFile("rules.json")
	}
	c.JSON(200, wh.Logic.Rules())
}
func (wh *WebHandler) GetRunRules(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")
	for _, rule := range wh.Logic.Rules() {
		if rule.Uuid() == id {
			switch action {
			case "enter":
				rule.RunEnter(wh.Logic.ActionProgressChan)
				c.JSON(200, "ok")
				return
			case "exit":
				rule.RunExit(wh.Logic.ActionProgressChan)
				c.JSON(200, "ok")
				return

			}
		}
	}
	c.String(404, "Rule not found")
}

func (wh *WebHandler) GetScheduleTasks(c *gin.Context) {
	if p := c.Request.URL.Query()["reload"]; len(p) > 0 {
		wh.Scheduler.Reload()
	}
	c.JSON(200, wh.Scheduler.Tasks())
}
func (wh *WebHandler) GetScheduleEntries(c *gin.Context) {
	c.JSON(200, wh.Scheduler.Cron.Entries())
}

func (wh *WebHandler) GetServerTrigger(c *gin.Context) {

	if node, ok := wh.Nodes.ByName("server").(*servernode.Node); ok {
		node.Set(c.Param("key"), c.Param("value"))
		wh.NodeServer.updateState(node.LogicChannel(), node)
		node.Reset(c.Param("key"))
		wh.NodeServer.updateState(node.LogicChannel(), node)
		c.JSON(200, node.State())
		return
	}
	c.String(500, "node server is wrong type")
}

func (wh *WebHandler) GetServerSet(c *gin.Context) {
	if node, ok := wh.Nodes.ByName("server").(*servernode.Node); ok {
		node.Set(c.Param("key"), c.Param("value"))
		wh.NodeServer.updateState(node.LogicChannel(), node)
		c.JSON(200, node.State())
		return
	}
	c.String(500, "node server is wrong type")
}
func (wh *WebHandler) GetDevices(c *gin.Context) {
	c.JSON(200, wh.Devices.AllWithState(wh.Nodes))
}
