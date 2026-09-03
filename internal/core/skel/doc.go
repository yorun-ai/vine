// Package skel implements generated Skel types, schemas, and registration.
//
// Vine currently uses the registered schema's skelc version to preserve the
// wire behavior of previously generated code. Schemas generated before skelc
// v0.15.0 encode nil slices and maps as null in JSON and CBOR. Schemas generated
// by skelc v0.15.0 or later use the current encoding behavior, which represents
// nil slices and maps as empty arrays and maps.
//
// TODO: Remove the pre-v0.15.0 encoding compatibility after it has remained
// available for several Vine releases and supported projects have regenerated
// their Skel code.
package skel
