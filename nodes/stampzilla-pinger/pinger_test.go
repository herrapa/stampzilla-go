package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPinger(t *testing.T) {
	target := &Target{
		Name: "localhost",
		Ip:   "127.0.0.1",
	}

	done := make(chan struct{})

	target.start(func(t *Target) {
		if done != nil {
			close(done)
			done = nil
		}
	})

	select {
	case <-done:
	case <-time.After(time.Second):
	}

	assert.Equal(t, true, target.Online)
}

func TestPinger100(t *testing.T) {
	target := &Target{
		Name: "localhost",
		Ip:   "192.0.2.1", // http://tools.ietf.org/html/rfc5737 - TEST-NET-1
	}
	target.Online = true

	done := make(chan struct{})

	target.start(func(t *Target) {
		if done != nil {
			close(done)
			done = nil
		}
	})

	select {
	case <-done:
	case <-time.After(time.Second):
	}

	assert.Equal(t, false, target.Online)
}
