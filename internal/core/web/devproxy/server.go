// Package devproxy serves a Web from a development server that announces itself
// in a state file, so a development run serves the frontend from its sources
// while the rest of the Web handler stays the generated one.
package devproxy

import (
	"encoding/json/v2"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/web/spec"
)

// stateReloadInterval is how long the state file a request read stays current: a
// development server that restarts on another port is followed within it. A test
// lowers it so one request reads the file again.
var stateReloadInterval = time.Second

// _State is what a development server writes about itself.
type _State struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Server forwards every request its Web receives to the development server named
// by a state file. The file carries the host and port a development run started
// the frontend on, so the frontend restarts on another port without a restart of
// the application.
//
// Embed Server by value in the Web handler so its Serve method belongs to the
// registered handler type, set the state file in DIInit, and delegate Routes to
// Server.Routes. A request the development server does not answer, and a state
// file that cannot be read, are reported as 502, because the Web has no content
// of its own to serve.
type Server struct {
	GinCtx *gin.Context `inject:""`

	stateFilePath string
}

// SetStateFile names the file the development server writes its host and port to.
func (s *Server) SetStateFile(stateFilePath string) {
	s.stateFilePath = stateFilePath
}

// Routes serves the mount root and every path below it, the same paths the
// embedded asset server serves: a mounted Web answers the request below its
// mount with the development server that was started for it.
func (s *Server) Routes(r *spec.Router) {
	if r.BasePath() != "/" {
		r.ANY("", s.Serve)
	}
	r.ANY("/*path", s.Serve)
}

// Serve forwards the request to the development server of the state file with
// the path the Web received, mount included, so a development server started for
// the Web's mount answers the route the client asked for.
func (s *Server) Serve() {
	followedServers.server(s.stateFilePath).serve(s.GinCtx)
}

// followedServers keeps the development server each state file names. A Web
// handler is built for every request, while the proxy and the address it forwards
// to outlive the request, so a frontend that restarts on another port is followed
// without a restart of the application.
var followedServers = &_FollowedServers{
	byStateFile: map[string]*_FollowedServer{},
}

// _FollowedServers keeps one development server per state file.
type _FollowedServers struct {
	mutex       sync.Mutex
	byStateFile map[string]*_FollowedServer
}

// _FollowedServer is the development server one state file names: the address it
// named last, and the proxy forwarding to it.
type _FollowedServer struct {
	stateFilePath string

	mutex    sync.Mutex
	loadedAt time.Time
	proxy    *httputil.ReverseProxy
	address  string
	problem  error
}

// server returns the development server following stateFilePath, and starts
// following the file when no request follows it yet.
func (s *_FollowedServers) server(stateFilePath string) *_FollowedServer {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	server := s.byStateFile[stateFilePath]
	if server == nil {
		server = &_FollowedServer{stateFilePath: stateFilePath}
		s.byStateFile[stateFilePath] = server
	}
	return server
}

// serve forwards one request, and answers 502 when the state file cannot be read
// or the development server does not answer it.
func (s *_FollowedServer) serve(ginCtx *gin.Context) {
	target, problem := s.target()
	if problem != nil {
		ginCtx.String(http.StatusBadGateway, problem.Error())
		return
	}
	target.ServeHTTP(ginCtx.Writer, ginCtx.Request)
}

// target returns the proxy forwarding to the development server the current state
// file names, reading the file again when the last read is older than
// stateReloadInterval. A state file that cannot be read keeps the last
// development server reachable: a file that is being rewritten, or one that was
// removed after the frontend stopped, is answered by the proxy already running.
func (s *_FollowedServer) target() (*httputil.ReverseProxy, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.proxy == nil || time.Since(s.loadedAt) >= stateReloadInterval {
		s.loadedAt = time.Now()
		state, err := readState(s.stateFilePath)
		s.problem = err
		if err == nil {
			s.setProxy(s.proxyFor(state), addressOf(state))
		}
	}
	if s.proxy == nil {
		return nil, s.problem
	}
	return s.proxy, nil
}

// proxyFor returns the proxy serving state, reusing the running one while it
// already points at the address the state file names.
func (s *_FollowedServer) proxyFor(state *_State) *httputil.ReverseProxy {
	if s.proxy != nil && s.address == addressOf(state) {
		return s.proxy
	}
	target := &url.URL{Scheme: "http", Host: addressOf(state)}
	logger.Info("forwarding to the development server",
		"target", target.String(), "state", s.stateFilePath)
	return developmentProxy(target)
}

// setProxy replaces the proxy a request forwards through, and the address it
// names. A request the proxy it replaces was still forwarding ends by itself: the
// development server it reached is the one that stopped or moved.
func (s *_FollowedServer) setProxy(replacement *httputil.ReverseProxy, address string) {
	if s.proxy == replacement {
		return
	}
	s.proxy = replacement
	s.address = address
}

// developmentProxy forwards a request to target with the path the Web received,
// mount included, so a development server started for the Web's mount answers the
// route the client asked for. The whole URL the Web handler read reaches the
// development server, escaping included, so a path the client encoded reaches it
// as the client sent it. A request the development server does not answer is
// reported as 502: the Web has no content of its own to serve.
func developmentProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.Host = target.Host
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			writer.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprintf(writer, "development server %s is not reachable", target.Host)
		},
	}
}

// addressOf is where a development server listens: a state file that names no
// host means the machine the application runs on.
func addressOf(state *_State) string {
	host := state.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(state.Port))
}

func readState(stateFilePath string) (*_State, error) {
	content, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("read development server state %s: %w", stateFilePath, err)
	}
	state := new(_State)
	err = json.Unmarshal(content, state)
	if err != nil {
		return nil, fmt.Errorf("parse development server state %s: %w", stateFilePath, err)
	}
	if state.Port <= 0 {
		return nil, fmt.Errorf("invalid development server port %d in %s", state.Port, stateFilePath)
	}
	return state, nil
}
