package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/stampzilla/stampzilla-go/protocol"
)

type ruleCondition struct {
	statePath  string
	comparator string
	value      string
}

type ruleAction struct {
	commands *protocol.Command
}

type rule struct {
	name       string
	conditions []*ruleCondition
	actions    []*ruleAction
}

type Logic struct {
	stateMap map[string]string // might not be needed
	rules    []*rule
	re       *regexp.Regexp
	sync.RWMutex
}

func NewLogic() *Logic {
	l := &Logic{stateMap: make(map[string]string)}
	l.re = regexp.MustCompile(`^([^\s\[][^\s\[]*)?(\[[0-9]+\])?$`)
	return l
}

func (l *Logic) AddRule(name string, cond []*ruleCondition, action []*ruleAction) {
	r := &rule{name, cond, action}
	l.Lock()
	l.rules = append(l.rules, r)
	l.Unlock()
}
func (l *Logic) ParseRules(states map[string]string) {
	for _, rule := range l.rules {
		l.parseRule(states, rule)
	}
}
func (l *Logic) parseRule(s map[string]string, r *rule) {

	for _, cond := range r.conditions {
		fmt.Println(cond.statePath)
		for _, state := range s {
			var test string
			err := l.path(state, cond.statePath, &test)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("path output:", test)
		}
	}

}

func (l *Logic) ListenForChanges() chan interface{} {
	//TODO maybe this should be a buffered channel so we dont block on send in netStart/newClient
	c := make(chan interface{})
	go l.listen(c)
	return c
}

// listen will run in a own goroutine and listen to incoming state changes and Parse them
func (l *Logic) listen(c chan interface{}) {
	for state := range c {
		l.ParseState(state)
	}
}

func (l *Logic) path(state string, jp string, t interface{}) error {
	var v interface{}
	err := json.Unmarshal([]byte(state), &v)
	if err != nil {
		return err
	}
	if jp == "" {
		return errors.New("invalid path")
	}
	for _, token := range strings.Split(jp, ".") {
		sl := l.re.FindAllStringSubmatch(token, -1)
		//fmt.Println("REGEXPtoken: ", token)
		//fmt.Println("REGEXP: ", sl)
		if len(sl) == 0 {
			return errors.New("invalid path1")
		}
		ss := sl[0]
		if ss[1] != "" {
			v = v.(map[string]interface{})[ss[1]]
		}
		if ss[2] != "" {
			i, err := strconv.Atoi(ss[2][1 : len(ss[2])-1])
			if err != nil {
				return errors.New("invalid path2")
			}
			v = v.([]interface{})[i]
		}
	}
	rt := reflect.ValueOf(t).Elem()
	//fmt.Println("RT:", rt)
	rv := reflect.ValueOf(v)
	//fmt.Println("RT:", rv)
	rt.Set(rv)
	return nil
}

func (l *Logic) ParseState(state interface{}) {
	//TODO parse all nodes.State here and generate something like this:
	// OR we dont use stateMap and only use rules Devices[2].On == true and parse it using jsonpath example below.
	// statemap["Devices[1].State"] = "OFF"
	// this might be usefull: http://play.golang.org/p/JQnry4s6KE
	// http://blog.golang.org/json-and-go
}

/*
Example of state:
State: {
	Devices: {
		1: {
			Id: "1",
			Name: "Dev1",
			State: "OFF",
			Type: ""
		},
		2: {
			Id: "2",
			Name: "Dev2",
			State: "ON",
			Type: ""
		}
	}
}
*/
