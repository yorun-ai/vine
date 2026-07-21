package assets

import (
	"bytes"
	"compress/gzip"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/util/vpre"
)

type _Encoding string

const (
	encodingNone _Encoding = "none"
	encodingGzip _Encoding = "gzip"
	encodingZstd _Encoding = "zstd"

	indexPath = "/index.html"
)

type _File struct {
	ModTime  time.Time
	Encoding _Encoding
	Content  []byte
}

type Accessor interface {
	Open(filename string, encodings []_Encoding) (*_File, bool)
}

type Server struct {
	GinCtx   *gin.Context `inject:""`
	accessor Accessor

	requestPath    string
	acceptHTML     bool
	acceptEncoding []_Encoding

	assetFile *_File
	assetOK   bool

	responseEncoding _Encoding
	responseContent  []byte
}

func NewServer(accessor Accessor) Server {
	return Server{accessor: accessor}
}

func (s *Server) SetContext(ginCtx *gin.Context) {
	s.GinCtx = ginCtx
}

func (s *Server) SetAccessor(accessor Accessor) {
	s.accessor = accessor
}

func (s *Server) Routes(r *spec.Router) {
	r.ANY("/*path", s.Serve)
}

func (s *Server) ServeAsset(ginCtx *gin.Context, accessor Accessor) {
	s.SetContext(ginCtx)
	s.SetAccessor(accessor)
	s.Serve()
}

func (s *Server) Serve() {
	if s.GinCtx.Request.Method != http.MethodGet && s.GinCtx.Request.Method != http.MethodHead {
		s.GinCtx.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	s.initRequest()
	s.loadFile()
	s.encodeResponse()
	s.writeResponse()
}

func (s *Server) initRequest() {
	s.requestPath = path.Clean("/" + s.GinCtx.Param("path"))
	if s.requestPath == "/" {
		s.requestPath = indexPath
	}

	s.acceptHTML = strings.Contains(s.GinCtx.GetHeader("Accept"), "text/html")
	s.acceptEncoding = acceptAssetEncodings(s.GinCtx.GetHeader("Accept-Encoding"))
}

func (s *Server) loadFile() {
	s.assetFile, s.assetOK = s.accessor.Open(s.requestPath, s.acceptEncoding)
	if s.assetOK || !s.acceptHTML {
		return
	}

	s.requestPath = indexPath
	s.assetFile, s.assetOK = s.accessor.Open(s.requestPath, s.acceptEncoding)
}

func (s *Server) writeResponse() {
	if !s.assetOK {
		s.GinCtx.AbortWithStatus(http.StatusNotFound)
		return
	}

	contentType := mime.TypeByExtension(path.Ext(s.requestPath))
	if contentType != "" {
		s.GinCtx.Writer.Header().Set("Content-Type", contentType)
	}

	if s.responseEncoding != encodingNone {
		s.GinCtx.Writer.Header().Set("Content-Encoding", string(s.responseEncoding))
		s.GinCtx.Writer.Header().Add("Vary", "Accept-Encoding")
	}

	http.ServeContent(
		s.GinCtx.Writer,
		s.GinCtx.Request,
		s.requestPath,
		s.assetFile.ModTime,
		bytes.NewReader(s.responseContent))
}

func acceptAssetEncodings(header string) []_Encoding {
	encodings := []_Encoding{}
	for _, item := range strings.Split(header, ",") {
		encoding, params, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil || params["q"] == "0" {
			continue
		}

		if encoding == string(encodingZstd) {
			encodings = append(encodings, encodingZstd)
		} else if encoding == string(encodingGzip) {
			encodings = append(encodings, encodingGzip)
		}
	}

	return encodings
}

func acceptsEncoding(encodings []_Encoding, encoding _Encoding) bool {
	for _, item := range encodings {
		if item == encoding {
			return true
		}
	}
	return false
}

func (s *Server) encodeResponse() {
	if !s.assetOK {
		return
	}

	s.responseEncoding = s.assetFile.Encoding
	s.responseContent = s.assetFile.Content
	if s.responseEncoding != encodingNone {
		return
	}

	switch preferredEncoding(s.acceptEncoding) {
	case encodingZstd:
		s.responseEncoding = encodingZstd
		s.responseContent = encodeZst(s.assetFile.Content)
	case encodingGzip:
		s.responseEncoding = encodingGzip
		s.responseContent = encodeGzip(s.assetFile.Content)
	}
}

func preferredEncoding(encodings []_Encoding) _Encoding {
	if acceptsEncoding(encodings, encodingZstd) {
		return encodingZstd
	}
	if acceptsEncoding(encodings, encodingGzip) {
		return encodingGzip
	}
	return encodingNone
}

func encodeZst(content []byte) []byte {
	var buffer bytes.Buffer
	writer, err := zstd.NewWriter(&buffer)
	vpre.CheckNilError(err, "open embedded static zstd writer failed")
	_, err = writer.Write(content)
	vpre.CheckNilError(err, "write embedded static zstd failed")
	err = writer.Close()
	vpre.CheckNilError(err, "close embedded static zstd failed")
	return buffer.Bytes()
}

func encodeGzip(content []byte) []byte {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(content)
	vpre.CheckNilError(err, "write embedded static gzip failed")
	err = writer.Close()
	vpre.CheckNilError(err, "close embedded static gzip failed")
	return buffer.Bytes()
}
