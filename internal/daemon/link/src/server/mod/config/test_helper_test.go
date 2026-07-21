package config

import (
	"context"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
)

func newTestReader(configValuesByName map[string]redised.ConfigValue) *Reader {
	valuesByKey := map[string]string{}
	for key, value := range configValuesByName {
		valuesByKey[redised.FormatConfigKey(key)] = marshalTestConfigValue(key, string(value.Value))
	}
	reader := &Reader{
		Context:   context.Background(),
		Client:    hubredis.NewClientForTest(valuesByKey),
		AppMinder: newTestMinder(),
	}
	reader.DIInit()
	return reader
}

func marshalTestConfigValue(name string, value string) string {
	return vcode.MustMarshalJsonS(redised.ConfigValue{
		Name:  name,
		Value: []byte(value),
	})
}

func newTestMinder() *minder.AppMinder {
	minder := &minder.AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		InprocFlag:            &app.InternalInprocFlag{Enabled: true},
		RegistryServiceClient: &_TestRegistryServiceClient{},
	}
	minder.DIInit()
	return minder
}

type _TestRegistryServiceClient struct{}

func (*_TestRegistryServiceClient) Register(hubskeled.AppRegistration, ...rpcclient.InvokeOption) {
}

func (*_TestRegistryServiceClient) Unregister(string, skel.UUID, ...rpcclient.InvokeOption) {
}

func (*_TestRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...rpcclient.InvokeOption) bool {
	return true
}

func newTestAppInstance(instanceID string) *minder.AppInstance {
	appInfo, err := meta.NewApp("demo.app", "1.0.0", instanceID)
	if err != nil {
		panic(err)
	}
	return &minder.AppInstance{
		AppInfo: appInfo,
	}
}

func registerTestAppInstance(reader *Reader, instanceID string) *minder.AppInstance {
	instance := newTestAppInstance(instanceID)
	reader.OnSetup(instance)
	return instance
}

func configValueSnapshot(reader *Reader, key string, appInstanceID string) string {
	value, _ := reader.findConfigValueSnapshot(appInstanceID, key)
	return value
}
