package mqtt

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	capmocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestInterface_Connected(t *testing.T) {
	t.Run("publisher is set correctly", func(t *testing.T) {
		i := Interface{}

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		err := i.Connected(context.Background(), m.Publish)
		assert.NoError(t, err)

		assert.NotNil(t, i.publisher)
	})

	t.Run("publishes capabilities if set to publish on connect", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		mapper.On("Gateways").Return(map[string]da.Gateway{"one": gw})

		capFlagOne := capabilities.HasProductInformationFlag
		capFlagTwo := capabilities.OnOffFlag

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capFlagOne, capFlagTwo},
		}

		gw.On("Devices").Return([]da.Device{d})

		hpi := &capmocks.HasProductInformation{}
		hpi.On("Name").Return("HasProductInformation")
		hpi.On("ProductInformation", mock.Anything, d).Return(capabilities.ProductInformation{
			Present: capabilities.Name,
			Name:    "Mock",
		}, nil)
		defer hpi.AssertExpectations(t)

		oo := &capmocks.OnOff{}
		oo.Mock.On("Name").Return("OnOff")
		oo.Mock.On("Status", mock.Anything, d).Return(true, nil)
		defer oo.AssertExpectations(t)

		gw.On("Capability", capFlagOne).Return(hpi)
		gw.On("Capability", capFlagTwo).Return(oo)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishStateOnConnect: true, PublishSummaryState: true}

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/HasProductInformation", d.DeviceIdentifier.String()), []byte{0x7b, 0x22, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x3a, 0x22, 0x4d, 0x6f, 0x63, 0x6b, 0x22, 0x7d}).Return(nil)
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/OnOff", d.DeviceIdentifier.String()), []byte{0x7b, 0x22, 0x53, 0x74, 0x61, 0x74, 0x65, 0x22, 0x3a, 0x74, 0x72, 0x75, 0x65, 0x7d}).Return(nil)

		err := i.Connected(context.Background(), m.Publish)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
	})
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, prefix string, payload []byte) error {
	args := m.Called(ctx, prefix, payload)
	return args.Error(0)
}
