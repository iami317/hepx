package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type ClientPluginOptions interface{}

type TypedClientPluginOptions struct {
	Type string `json:"type"`
	ClientPluginOptions
}

func (c *TypedClientPluginOptions) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return nil
	}

	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	if c.Type == "" {
		return errors.New("plugin type is empty")
	}

	v, ok := clientPluginOptionsTypeMap[typeStruct.Type]
	if !ok {
		return fmt.Errorf("unknown plugin type: %s", typeStruct.Type)
	}
	options := reflect.New(v).Interface().(ClientPluginOptions)

	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(options); err != nil {
		return fmt.Errorf("unmarshal ClientPluginOptions error: %v", err)
	}
	c.ClientPluginOptions = options
	return nil
}

func (c *TypedClientPluginOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ClientPluginOptions)
}

const (
	PluginHTTP2HTTPS       = "http2https"
	PluginHTTPProxy        = "http_proxy"
	PluginHTTPS2HTTP       = "https2http"
	PluginHTTPS2HTTPS      = "https2https"
	PluginSocks5           = "socks5"
	PluginStaticFile       = "static_file"
	PluginUnixDomainSocket = "unix_domain_socket"
)

var clientPluginOptionsTypeMap = map[string]reflect.Type{
	PluginHTTP2HTTPS:       reflect.TypeOf(HTTP2HTTPSPluginOptions{}),
	PluginHTTPProxy:        reflect.TypeOf(HTTPProxyPluginOptions{}),
	PluginHTTPS2HTTP:       reflect.TypeOf(HTTPS2HTTPPluginOptions{}),
	PluginHTTPS2HTTPS:      reflect.TypeOf(HTTPS2HTTPSPluginOptions{}),
	PluginSocks5:           reflect.TypeOf(Socks5PluginOptions{}),
	PluginStaticFile:       reflect.TypeOf(StaticFilePluginOptions{}),
	PluginUnixDomainSocket: reflect.TypeOf(UnixDomainSocketPluginOptions{}),
}

type HTTP2HTTPSPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
}

type HTTPProxyPluginOptions struct {
	Type         string `json:"type,omitempty"`
	HTTPUser     string `json:"httpUser,omitempty"`
	HTTPPassword string `json:"httpPassword,omitempty"`
}

type HTTPS2HTTPPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
	CrtPath           string           `json:"crtPath,omitempty"`
	KeyPath           string           `json:"keyPath,omitempty"`
}

type HTTPS2HTTPSPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
	CrtPath           string           `json:"crtPath,omitempty"`
	KeyPath           string           `json:"keyPath,omitempty"`
}

type Socks5PluginOptions struct {
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type StaticFilePluginOptions struct {
	Type         string `json:"type,omitempty"`
	LocalPath    string `json:"localPath,omitempty"`
	StripPrefix  string `json:"stripPrefix,omitempty"`
	HTTPUser     string `json:"httpUser,omitempty"`
	HTTPPassword string `json:"httpPassword,omitempty"`
}

type UnixDomainSocketPluginOptions struct {
	Type     string `json:"type,omitempty"`
	UnixPath string `json:"unixPath,omitempty"`
}
