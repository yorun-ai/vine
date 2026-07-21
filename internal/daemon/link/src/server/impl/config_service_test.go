package impl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/config"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
)

func TestConfigServiceReturnsInstantValue(t *testing.T) {
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	assert.NoError(t, err)
	trace := meta.InitialTrace()
	actor := meta.NewAbsentActor()
	assert.NoError(t, err)

	appMinder := &minder.AppMinder{}
	appMinder.DIInit()
	reader := &config.Reader{
		Context: context.Background(),
		Client: hubredis.NewClientForTest(map[string]string{
			redised.FormatConfigKey("demo.FeatureConfig"): marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
		}),
		AppMinder: appMinder,
	}
	reader.DIInit()
	reader.OnSetup(&minder.AppInstance{
		AppInfo: app,
	})
	service := &ConfigServiceServerImpl{
		Reader:  reader,
		Context: spec.NewContext(context.Background(), trace, app, nil, actor),
	}

	value := service.GetInstant("demo.FeatureConfig")

	assert.Equal(t, `{"enabled":true}`, value)
	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(app.InstanceId(), "demo.FeatureConfig"))
}

func TestConfigServiceReturnsEternalValue(t *testing.T) {
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	assert.NoError(t, err)
	trace := meta.InitialTrace()
	actor := meta.NewAbsentActor()
	assert.NoError(t, err)

	appMinder := &minder.AppMinder{}
	appMinder.DIInit()
	reader := &config.Reader{
		Context: context.Background(),
		Client: hubredis.NewClientForTest(map[string]string{
			redised.FormatConfigKey("demo.FeatureConfig"): marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
		}),
		AppMinder: appMinder,
	}
	reader.DIInit()
	reader.OnSetup(&minder.AppInstance{
		AppInfo: app,
	})
	service := &ConfigServiceServerImpl{
		Reader:  reader,
		Context: spec.NewContext(context.Background(), trace, app, nil, actor),
	}

	value := service.GetEternal("demo.FeatureConfig")

	assert.Equal(t, `{"enabled":true}`, value)
	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(app.InstanceId(), "demo.FeatureConfig"))
}

func marshalTestConfigValue(name string, value string) string {
	return vcode.MustMarshalJsonS(redised.ConfigValue{
		Name:  name,
		Value: []byte(value),
	})
}
