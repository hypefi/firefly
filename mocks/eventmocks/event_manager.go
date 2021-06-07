// Code generated by mockery v1.0.0. DO NOT EDIT.

package eventmocks

import (
	context "context"

	blockchain "github.com/kaleido-io/firefly/pkg/blockchain"

	fftypes "github.com/kaleido-io/firefly/pkg/fftypes"

	mock "github.com/stretchr/testify/mock"
)

// EventManager is an autogenerated mock type for the EventManager type
type EventManager struct {
	mock.Mock
}

// BatchPinComplete provides a mock function with given fields: bi, batch, signingIdentity, protocolTxID, additionalInfo
func (_m *EventManager) BatchPinComplete(bi blockchain.Plugin, batch *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	ret := _m.Called(bi, batch, signingIdentity, protocolTxID, additionalInfo)

	var r0 error
	if rf, ok := ret.Get(0).(func(blockchain.Plugin, *blockchain.BatchPin, string, string, fftypes.JSONObject) error); ok {
		r0 = rf(bi, batch, signingIdentity, protocolTxID, additionalInfo)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateDurableSubscription provides a mock function with given fields: ctx, subDef
func (_m *EventManager) CreateDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) error {
	ret := _m.Called(ctx, subDef)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *fftypes.Subscription) error); ok {
		r0 = rf(ctx, subDef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteDurableSubscription provides a mock function with given fields: ctx, subDef
func (_m *EventManager) DeleteDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) error {
	ret := _m.Called(ctx, subDef)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *fftypes.Subscription) error); ok {
		r0 = rf(ctx, subDef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletedSubscriptions provides a mock function with given fields:
func (_m *EventManager) DeletedSubscriptions() chan<- *fftypes.UUID {
	ret := _m.Called()

	var r0 chan<- *fftypes.UUID
	if rf, ok := ret.Get(0).(func() chan<- *fftypes.UUID); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan<- *fftypes.UUID)
		}
	}

	return r0
}

// NewEvents provides a mock function with given fields:
func (_m *EventManager) NewEvents() chan<- int64 {
	ret := _m.Called()

	var r0 chan<- int64
	if rf, ok := ret.Get(0).(func() chan<- int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan<- int64)
		}
	}

	return r0
}

// NewSubscriptions provides a mock function with given fields:
func (_m *EventManager) NewSubscriptions() chan<- *fftypes.UUID {
	ret := _m.Called()

	var r0 chan<- *fftypes.UUID
	if rf, ok := ret.Get(0).(func() chan<- *fftypes.UUID); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan<- *fftypes.UUID)
		}
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *EventManager) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TransactionUpdate provides a mock function with given fields: bi, txTrackingID, txState, protocolTxID, errorMessage, additionalInfo
func (_m *EventManager) TransactionUpdate(bi blockchain.Plugin, txTrackingID string, txState fftypes.OpStatus, protocolTxID string, errorMessage string, additionalInfo fftypes.JSONObject) error {
	ret := _m.Called(bi, txTrackingID, txState, protocolTxID, errorMessage, additionalInfo)

	var r0 error
	if rf, ok := ret.Get(0).(func(blockchain.Plugin, string, fftypes.OpStatus, string, string, fftypes.JSONObject) error); ok {
		r0 = rf(bi, txTrackingID, txState, protocolTxID, errorMessage, additionalInfo)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WaitStop provides a mock function with given fields:
func (_m *EventManager) WaitStop() {
	_m.Called()
}
