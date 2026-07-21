package meta

import "net/netip"

type Initiator interface {
	App
	Dialer() string
	IpAddr() string
}

type _Initiator struct {
	_App
	ipAddr string
	dialer string
}

func NewInitiator(name string, version string, instanceId string, dialer string, ipAddr string) (Initiator, error) {
	appInfo, err := NewApp(name, version, instanceId)
	if err != nil {
		return nil, err
	}

	if ipAddr != "" {
		if _, err := netip.ParseAddr(ipAddr); err != nil {
			return nil, err
		}
	}

	return &_Initiator{
		_App: _App{
			name:       appInfo.Name(),
			version:    appInfo.Version(),
			instanceId: appInfo.InstanceId(),
		},
		ipAddr: ipAddr,
		dialer: dialer,
	}, nil
}

func (i *_Initiator) Dialer() string {
	return i.dialer
}

func (i *_Initiator) IpAddr() string {
	return i.ipAddr
}

type _InitiatorPayload struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	InstanceId string `json:"instanceId"`
	Dialer     string `json:"dialer"`
	IpAddr     string `json:"ipAddr"`
}

func DecodeInitiatorFromBase64(value string) (Initiator, error) {
	if value == "" {
		return nil, nil
	}

	payload, err := decodePayloadFromBase64[*_InitiatorPayload](value)
	if err != nil {
		return nil, err
	}
	return NewInitiator(payload.Name, payload.Version, payload.InstanceId, payload.Dialer, payload.IpAddr)
}

func EncodeInitiatorToBase64(initiator Initiator) string {
	return encodePayloadToBase64(&_InitiatorPayload{
		Name:       initiator.Name(),
		Version:    initiator.Version(),
		InstanceId: initiator.InstanceId(),
		Dialer:     initiator.Dialer(),
		IpAddr:     initiator.IpAddr(),
	})
}
