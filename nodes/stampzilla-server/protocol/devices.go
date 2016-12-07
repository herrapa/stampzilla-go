package protocol

import (
	"sync"

	"github.com/stampzilla/stampzilla-go/protocol/devices"
)

type Devices struct {
	devices devices.Map
	sync.RWMutex
}

func NewDevices() *Devices {
	n := &Devices{}
	n.devices = make(devices.Map)
	return n
}

func (n *Devices) ByUuid(uuid string) *devices.Device {
	n.RLock()
	defer n.RUnlock()
	if node, ok := n.devices[uuid]; ok {
		return node
	}
	return nil
}
func (n *Devices) All() map[string]*devices.Device {
	n.RLock()
	defer n.RUnlock()
	return n.devices
}
func (n *Devices) ShallowCopy() devices.Map {
	n.RLock()
	defer n.RUnlock()
	r := make(map[string]*devices.Device)
	for k, v := range n.devices {
		copy := *v
		r[k] = &copy
	}
	return r
}
func (n *Devices) AllWithState(nodes *Nodes) devices.Map {
	devices := n.ShallowCopy()
	for _, device := range devices {
		node := nodes.ByUuid(device.Node)
		if node == nil {
			device.State = nil
			continue //node is offline and we dont have the state
		}
		device.SyncState(node.State())
	}
	return devices
}
func (n *Devices) Add(nodeUuid string, device *devices.Device) error {
	n.Lock()
	defer n.Unlock()
	n.devices[nodeUuid+"."+device.Id] = device
	return nil
}
func (n *Devices) Delete(uuid string) {
	n.Lock()
	defer n.Unlock()
	delete(n.devices, uuid)
}

//SetOfflineByNode marks device from a single node offline and returns a list of all marked devices
func (n *Devices) SetOfflineByNode(nodeUUID string) (list []*devices.Device) {
	n.Lock()
	defer n.Unlock()

	list = make([]*devices.Device, 0)
	for _, dev := range n.devices {
		if dev.Node == nodeUUID {
			dev.Online = false
			list = append(list, dev)
		}
	}

	return
}
