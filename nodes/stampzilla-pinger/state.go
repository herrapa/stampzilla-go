package main

import (
	"github.com/stampzilla/stampzilla-go/nodes/basenode"
	"github.com/stampzilla/stampzilla-go/protocol"
)

var state targetState

type targetState struct {
	Targets    map[string]*Target
	connection basenode.Connection
}

func (s *targetState) add(t *Target) {
	s.Targets[t.Name] = t

	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     t.Name,
		Feedback: `Targets["` + t.Name + `"].Online`,
	})

	node.AddElement(&protocol.Element{
		Type:     protocol.ElementTypeText,
		Name:     t.Name,
		Feedback: `Targets["` + t.Name + `"].Lag`,
	})

	t.start(func(t *Target) {
		s.connection.Send(node.Node())
	})
}
