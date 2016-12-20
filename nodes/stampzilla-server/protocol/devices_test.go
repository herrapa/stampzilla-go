package protocol

import (
	"testing"

	"github.com/stampzilla/stampzilla-go/protocol/devices"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	devs := NewDevices()
	device := devices.NewDevice()
	device.Id = "devuuid"
	devs.Add("nodeuuid", device)
	assert.Contains(t, devs.devices, "nodeuuid.devuuid")
}

func TestShallowCopy(t *testing.T) {
	devs := NewDevices()
	device := devices.NewDevice()
	device.Id = "devuuid"
	device.Name = "Name"
	devs.Add("nodeuuid", device)

	copied := devs.ShallowCopy()
	assert.Contains(t, copied, "nodeuuid.devuuid")
	device.Name = "NameChange"
	assert.Equal(t, "Name", copied["nodeuuid.devuuid"].Name)
	assert.Equal(t, "NameChange", devs.devices["nodeuuid.devuuid"].Name)
}

func TestSetOfflineByNode(t *testing.T) {
	devs := NewDevices()

	device1 := devices.NewDevice()
	device1.Id = "1"
	device1.Node = "node1"
	device1.Online = true

	device2 := devices.NewDevice()
	device2.Id = "2"
	device2.Node = "node2"
	device2.Online = true

	devs.Add("node1", device1)
	devs.Add("node2", device2)

	list := devs.SetOfflineByNode("node1")

	assert.Len(t, list, 1)
	assert.Equal(t, false, device1.Online)
}