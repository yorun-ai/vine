package skel

import (
	"encoding/json/v2"

	"github.com/Masterminds/semver/v3"
	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

const nilContainersAsEmptySince = "v0.15.0"

type GeneratedInfo struct {
	CompilerVersion string `json:"compilerVersion"`
}

var legacyEncoder = newLegacyEncoder()

func newLegacyEncoder() vcode.Encoder {
	jsonOptions := json.JoinOptions(
		json.FormatNilSliceAsNull(true),
		json.FormatNilMapAsNull(true),
	)
	cborMode, err := (cbor.EncOptions{}).EncMode()
	vpre.MustNil(err)
	return vcode.NewEncoder(jsonOptions, cborMode)
}

func encoderForGeneratedInfo(generated *GeneratedInfo) vcode.Encoder {
	if generated.CompilerVersion == buildinfo.DevVersion {
		return vcode.DefaultEncoder()
	}
	compilerVersion := semver.MustParse(generated.CompilerVersion)
	if compilerVersion.Compare(semver.MustParse(nilContainersAsEmptySince)) < 0 {
		return legacyEncoder
	}
	return vcode.DefaultEncoder()
}
