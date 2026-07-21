package skel

type AuthMode string

const (
	AuthModeUnset  AuthMode = "unset"
	AuthModeAuth   AuthMode = "auth"
	AuthModeNoAuth AuthMode = "noauth"
)
