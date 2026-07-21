package embedded

import (
	"errors"
	"strconv"
	"strings"

	"github.com/tidwall/redcon"
)

func (s *Store) handleCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) == 0 {
		conn.WriteError("ERR empty command")
		return
	}
	command := strings.ToUpper(string(cmd.Args[0]))
	if command != "HELLO" && !canRunCommand(connContext(conn).role, command) {
		if connContext(conn).role == userRoleNone {
			conn.WriteError("NOAUTH Authentication required.")
			return
		}
		conn.WriteError("NOPERM this user has no permissions to run the '" + strings.ToLower(command) + "' command")
		return
	}
	switch command {
	case "HELLO":
		s.handleHello(conn, cmd)
	case "PING":
		s.handlePing(conn, cmd)
	case "GET":
		s.handleGet(conn, cmd)
	case "SCAN":
		s.handleScan(conn, cmd)
	case "TTL":
		s.handleTTL(conn, cmd)
	case "SUBSCRIBE":
		s.handleSubscribe(conn, cmd, false)
	case "UNSUBSCRIBE":
		conn.WriteArray(3)
		conn.WriteBulkString("unsubscribe")
		conn.WriteNull()
		conn.WriteInt(0)
	case "PSUBSCRIBE":
		s.handleSubscribe(conn, cmd, true)
	case "PUNSUBSCRIBE":
		conn.WriteArray(3)
		conn.WriteBulkString("punsubscribe")
		conn.WriteNull()
		conn.WriteInt(0)
	case "SET":
		s.handleSet(conn, cmd)
	case "INCR":
		s.handleIncr(conn, cmd)
	case "DEL":
		s.handleDel(conn, cmd)
	case "PUBLISH":
		s.handlePublish(conn, cmd)
	case "EXPIRE":
		s.handleExpire(conn, cmd)
	default:
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
	}
}

func (s *Store) handleHello(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) == 1 {
		s.writeHello(conn)
		return
	}
	proto, err := strconv.Atoi(string(cmd.Args[1]))
	if err != nil || proto != 2 {
		conn.WriteError("NOPROTO unsupported protocol version")
		return
	}
	if len(cmd.Args) == 2 {
		s.writeHello(conn)
		return
	}
	if len(cmd.Args) != 5 || !strings.EqualFold(string(cmd.Args[2]), "AUTH") {
		conn.WriteError("ERR syntax error")
		return
	}
	role, ok := s.authenticate(string(cmd.Args[3]), string(cmd.Args[4]))
	if !ok {
		conn.WriteError("WRONGPASS invalid username-password pair or user is disabled.")
		return
	}
	conn.SetContext(&_ConnContext{role: role})
	s.writeHello(conn)
}

func (s *Store) writeHello(conn redcon.Conn) {
	conn.WriteArray(6)
	conn.WriteBulkString("server")
	conn.WriteBulkString("redis")
	conn.WriteBulkString("version")
	conn.WriteBulkString("7.0.0")
	conn.WriteBulkString("proto")
	conn.WriteInt(2)
}

func (s *Store) handlePing(conn redcon.Conn, cmd redcon.Command) {
	switch len(cmd.Args) {
	case 1:
		conn.WriteString("PONG")
	case 2:
		conn.WriteBulk(cmd.Args[1])
	default:
		conn.WriteError("ERR wrong number of arguments for 'ping' command")
	}
}

func (s *Store) handleGet(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for 'get' command")
		return
	}
	value, ok := s.Get(string(cmd.Args[1]))
	if !ok {
		conn.WriteNull()
		return
	}
	conn.WriteBulkString(value)
}

func (s *Store) handleScan(conn redcon.Conn, cmd redcon.Command) {
	option, err := parseScanOption(cmd.Args)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	keys, cursor, err := s.ScanKeys(option.cursor, option.match, option.count)
	if err != nil {
		conn.WriteError("ERR " + err.Error())
		return
	}
	conn.WriteArray(2)
	conn.WriteBulkString(strconv.FormatUint(cursor, 10))
	conn.WriteArray(len(keys))
	for _, key := range keys {
		conn.WriteBulkString(key)
	}
}

func (s *Store) handleTTL(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for 'ttl' command")
		return
	}
	conn.WriteInt(s.TTL(string(cmd.Args[1])))
}

func (s *Store) handleSubscribe(conn redcon.Conn, cmd redcon.Command, pattern bool) {
	if len(cmd.Args) < 2 {
		conn.WriteError("ERR wrong number of arguments for '" + strings.ToLower(string(cmd.Args[0])) + "' command")
		return
	}
	for i := 1; i < len(cmd.Args); i++ {
		s.Subscribe(conn, string(cmd.Args[i]), pattern)
	}
}

func (s *Store) handleSet(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for 'set' command")
		return
	}
	s.Set(string(cmd.Args[1]), string(cmd.Args[2]))
	conn.WriteString("OK")
}

func (s *Store) handleIncr(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for 'incr' command")
		return
	}
	value, err := s.Incr(string(cmd.Args[1]))
	if err != nil {
		conn.WriteError("ERR value is not an integer or out of range")
		return
	}
	conn.WriteInt64(value)
}

func (s *Store) handleDel(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) < 2 {
		conn.WriteError("ERR wrong number of arguments for 'del' command")
		return
	}
	deleted := 0
	for i := 1; i < len(cmd.Args); i++ {
		if s.Del(string(cmd.Args[i])) {
			deleted++
		}
	}
	conn.WriteInt(deleted)
}

func (s *Store) handlePublish(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for 'publish' command")
		return
	}
	conn.WriteInt(s.Publish(string(cmd.Args[1]), string(cmd.Args[2])))
}

func (s *Store) handleExpire(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for 'expire' command")
		return
	}
	seconds, err := strconv.Atoi(string(cmd.Args[2]))
	if err != nil || seconds <= 0 {
		conn.WriteError("ERR invalid expire time")
		return
	}
	if s.Expire(string(cmd.Args[1]), seconds) {
		conn.WriteInt(1)
		return
	}
	conn.WriteInt(0)
}

func parseScanOption(args [][]byte) (_ScanOption, error) {
	if len(args) < 2 {
		return _ScanOption{}, errors.New("ERR wrong number of arguments for 'scan' command")
	}
	cursor, err := strconv.ParseUint(string(args[1]), 10, 64)
	if err != nil {
		return _ScanOption{}, errors.New("ERR invalid scan cursor")
	}

	option := _ScanOption{
		cursor: cursor,
		match:  "*",
		count:  scanDefaultCount,
	}
	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return _ScanOption{}, errors.New("ERR syntax error")
		}
		switch strings.ToUpper(string(args[i])) {
		case "MATCH":
			option.match = string(args[i+1])
		case "COUNT":
			count, err := strconv.Atoi(string(args[i+1]))
			if err != nil || count <= 0 {
				return _ScanOption{}, errors.New("ERR invalid scan count")
			}
			option.count = count
		default:
			return _ScanOption{}, errors.New("ERR syntax error")
		}
	}
	return option, nil
}
