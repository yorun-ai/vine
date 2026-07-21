package embedded

import (
	"crypto/subtle"
	"strings"
)

const hubServerUsername = "hubserver"

type _UserRole int

const (
	userRoleNone _UserRole = iota
	userRoleClient
	userRoleServer
)

type _ConnContext struct {
	role _UserRole
}

var clientCommands = map[string]struct{}{
	"HELLO":        {},
	"PING":         {},
	"GET":          {},
	"SCAN":         {},
	"TTL":          {},
	"SUBSCRIBE":    {},
	"UNSUBSCRIBE":  {},
	"PSUBSCRIBE":   {},
	"PUNSUBSCRIBE": {},
}

func connContext(conn interface{ Context() interface{} }) *_ConnContext {
	ctx, ok := conn.Context().(*_ConnContext)
	if !ok {
		return &_ConnContext{role: userRoleClient}
	}
	return ctx
}

func (s *Store) authenticate(username string, password string) (_UserRole, bool) {
	passwordMatches := subtle.ConstantTimeCompare([]byte(password), []byte(s.serverPassword)) == 1
	if username == hubServerUsername && passwordMatches {
		return userRoleServer, true
	}
	return userRoleNone, false
}

func canRunCommand(role _UserRole, command string) bool {
	if role == userRoleServer {
		return true
	}
	if role == userRoleClient {
		_, ok := clientCommands[strings.ToUpper(command)]
		return ok
	}
	return false
}
