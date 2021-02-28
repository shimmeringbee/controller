package lighting

import "github.com/shimmeringbee/controller/metadata"

func New(deviceOrganiser metadata.DeviceOrganiser) {

}

type Application struct {
}

func (a *Application) AttachToMux() chan interface{} {
	return nil
}
