package logic

import "encoding/json"
import serverprotocol "github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/protocol"

//type Actions interface {
//Run()
//GetByUuid(string) Action
//Actions() []Action
//}

type Actions struct {
	Actions_ []*action             `json:"actions"`
	Nodes    *serverprotocol.Nodes `json:"-" inject:""`
}

func NewActions() *Actions {
	return &Actions{}
}

func (a *Actions) Run() {
	for _, v := range a.Actions_ {
		v.Run()
	}
}
func (a *Actions) Actions() []*action {
	return a.Actions_
}
func (a *Actions) Start() {
	mapper := newActionsMapper()
	mapper.Load(a)
}

func (a *Actions) GetByUuid(uuid string) Action {
	for _, v := range a.Actions_ {
		if v.Uuid() == uuid {
			return v
		}

	}
	return nil
}

func (a *Actions) UnmarshalJSON(b []byte) (err error) {
	type localActions Actions
	la := localActions{}
	if err = json.Unmarshal(b, &la); err == nil {
		for _, action := range la.Actions_ {
			action.SetNodes(a.Nodes)
			for _, c := range action.Commands {
				c.nodes = a.Nodes
			}
			a.Actions_ = append(a.Actions_, action)
		}
		return
	}
	return
}
