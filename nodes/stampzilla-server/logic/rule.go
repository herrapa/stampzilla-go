package logic

import (
	"sync"

	log "github.com/cihub/seelog"
	"github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/protocol"
)

type Rule interface {
	Uuid() string
	Name() string
	SetUuid(string)
	CondState() bool
	SetCondState(bool)
	RunEnter()
	RunExit()
	AddExitAction(Action)
	AddEnterAction(Action)
	AddCondition(RuleCondition)
	Conditions() []RuleCondition
}

type rule struct {
	Name_         string          `json:"name"`
	Uuid_         string          `json:"uuid"`
	Conditions_   []RuleCondition `json:"conditions"`
	EnterActions_ []string        `json:"enterActions"`
	ExitActions_  []string        `json:"exitActions"`
	enterActions_ []Action
	exitActions_  []Action
	condState     bool
	sync.RWMutex
	nodes *protocol.Nodes
}

func (r *rule) Uuid() string {
	r.RLock()
	defer r.RUnlock()
	return r.Uuid_
}
func (r *rule) Name() string {
	r.RLock()
	defer r.RUnlock()
	return r.Name_
}
func (r *rule) SetUuid(uuid string) {
	r.Lock()
	r.Uuid_ = uuid
	r.Unlock()
}
func (r *rule) CondState() bool {
	r.RLock()
	defer r.RUnlock()
	return r.condState
}
func (r *rule) Conditions() []RuleCondition {
	r.RLock()
	defer r.RUnlock()
	return r.Conditions_
}
func (r *rule) SetCondState(cond bool) {
	r.RLock()
	r.condState = cond
	r.RUnlock()
}
func (r *rule) RunEnter() {
	for _, a := range r.exitActions_ {
		a.Cancel()
	}
	for _, a := range r.enterActions_ {
		a.Run()
	}
}
func (r *rule) RunExit() {
	for _, a := range r.enterActions_ {
		a.Cancel()
	}
	for _, a := range r.exitActions_ {
		a.Run()
	}
}
func (r *rule) AddExitAction(a Action) {
	if a == nil {
		log.Error("Error adding ExitAction. Action is nil")
		return
	}
	r.Lock()
	r.exitActions_ = append(r.exitActions_, a)
	r.ExitActions_ = append(r.ExitActions_, a.Uuid())
	r.Unlock()
}
func (r *rule) AddEnterAction(a Action) {
	if a == nil {
		log.Error("Error adding EnterAction. Action is nil")
		return
	}
	r.Lock()
	r.enterActions_ = append(r.enterActions_, a)
	r.EnterActions_ = append(r.EnterActions_, a.Uuid())
	r.Unlock()
}
func (r *rule) AddCondition(a RuleCondition) {
	r.Lock()
	r.Conditions_ = append(r.Conditions_, a)
	r.Unlock()
}
