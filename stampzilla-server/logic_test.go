package main

import (
	"testing"

	"github.com/stampzilla/stampzilla-go/protocol"
)

func TestParseRuleState(t *testing.T) {

	logic := NewLogic()

	conditions := []*ruleCondition{&ruleCondition{"Devices.1.State", "==", "ON"}}
	actions := []*ruleAction{&ruleAction{&protocol.Command{"testcommand", nil}}}
	logic.AddRule("test rule 1", conditions, actions)

	state := `
		{
			"Devices": {
				"1": {
					"Id": "1",
					"Name": "Dev1",
					"State": "OFF",
					"Type": ""
				},
				"2": {
					"Id": "2",
					"Name": "Dev2",
					"State": "ON",
					"Type": ""
				}
			}
		}
	`

	fakeStates := make(map[string]string)
	fakeStates["uuidasdfasdf"] = state

	logic.ParseRules(fakeStates)

	//t.Errorf("OutputValue wrong expected: %s got %s", 55, p.OutputValue())
}
