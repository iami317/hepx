package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/iami317/hepx/pkg/config/types"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	"github.com/iami317/hepx/pkg/config/v1/validation"
)

// WordSepNormalizeFunc changes all flags that contain "_" separators
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
	}
	return pflag.NormalizedName(name)
}

type RegisterFlagOption func(*registerFlagOptions)

type registerFlagOptions struct {
	sshMode bool
}

func WithSSHMode() RegisterFlagOption {
	return func(o *registerFlagOptions) {
		o.sshMode = true
	}
}

type BandwidthQuantityFlag struct {
	V *types.BandwidthQuantity
}

func (f *BandwidthQuantityFlag) Set(s string) error {
	return f.V.UnmarshalString(s)
}

func (f *BandwidthQuantityFlag) String() string {
	return f.V.String()
}

func (f *BandwidthQuantityFlag) Type() string {
	return "string"
}

func RegisterProxyFlags(cmd *cobra.Command, c v1.ProxyConfigurer, opts ...RegisterFlagOption) {
	registerProxyBaseConfigFlags(cmd, c.GetBaseConfig(), opts...)

	switch cc := c.(type) {
	case *v1.TCPProxyConfig:
		cmd.Flags().IntVarP(&cc.RemotePort, "remote_port", "r", 0, "远程端口")
	case *v1.UDPProxyConfig:
		cmd.Flags().IntVarP(&cc.RemotePort, "remote_port", "r", 0, "远程端口")
	case *v1.HTTPProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
		cmd.Flags().StringSliceVarP(&cc.Locations, "locations", "", []string{}, "地点")
		cmd.Flags().StringVarP(&cc.HTTPUser, "http_user", "", "", "HTTP 身份验证用户")
		cmd.Flags().StringVarP(&cc.HTTPPassword, "http_pwd", "", "", "HTTP 身份验证密码")
		cmd.Flags().StringVarP(&cc.HostHeaderRewrite, "host_header_rewrite", "", "", "主机标头重写")
	case *v1.HTTPSProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
	case *v1.TCPMuxProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
		cmd.Flags().StringVarP(&cc.Multiplexer, "mux", "", "", "复用")
		cmd.Flags().StringVarP(&cc.HTTPUser, "http_user", "", "", "HTTP 身份验证用户")
		cmd.Flags().StringVarP(&cc.HTTPPassword, "http_pwd", "", "", "HTTP 身份验证密码")
	case *v1.STCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "密钥")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "允许访客用户")
	case *v1.SUDPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "密钥")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "允许访客用户")
	case *v1.XTCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "密钥")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "允许访客用户")
	}
}

func registerProxyBaseConfigFlags(cmd *cobra.Command, c *v1.ProxyBaseConfig, opts ...RegisterFlagOption) {
	if c == nil {
		return
	}
	options := &registerFlagOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cmd.Flags().StringVarP(&c.Name, "proxy_name", "n", "", "代理名称")

	if !options.sshMode {
		cmd.Flags().StringVarP(&c.LocalIP, "local_ip", "i", "127.0.0.1", "本地 IP")
		cmd.Flags().IntVarP(&c.LocalPort, "local_port", "l", 0, "本地端口")
		cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "使用加密")
		cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "使用压缩")
		cmd.Flags().StringVarP(&c.Transport.BandwidthLimitMode, "bandwidth_limit_mode", "", types.BandwidthLimitModeClient, "带宽限制模式")
		cmd.Flags().VarP(&BandwidthQuantityFlag{V: &c.Transport.BandwidthLimit}, "bandwidth_limit", "", "带宽限制（例如 100KB 或 1MB）")
	}
}

func registerProxyDomainConfigFlags(cmd *cobra.Command, c *v1.DomainConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringSliceVarP(&c.CustomDomains, "custom_domain", "d", []string{}, "自定义域名")
	cmd.Flags().StringVarP(&c.SubDomain, "sd", "", "", "子域名")
}

func RegisterVisitorFlags(cmd *cobra.Command, c v1.VisitorConfigurer, opts ...RegisterFlagOption) {
	registerVisitorBaseConfigFlags(cmd, c.GetBaseConfig(), opts...)

	// add visitor flags if exist
}

func registerVisitorBaseConfigFlags(cmd *cobra.Command, c *v1.VisitorBaseConfig, _ ...RegisterFlagOption) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "visitor_name", "n", "", "访客姓名")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "使用加密")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "使用压缩")
	cmd.Flags().StringVarP(&c.SecretKey, "sk", "", "", "密钥")
	cmd.Flags().StringVarP(&c.ServerName, "server_name", "", "", "服务器名称")
	cmd.Flags().StringVarP(&c.BindAddr, "bind_addr", "", "", "绑定 ADDR")
	cmd.Flags().IntVarP(&c.BindPort, "bind_port", "", 0, "绑定端口")
}

func RegisterClientCommonConfigFlags(cmd *cobra.Command, c *v1.ClientCommonConfig, opts ...RegisterFlagOption) {
	options := &registerFlagOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if !options.sshMode {
		cmd.PersistentFlags().StringVarP(&c.ServerAddr, "server_addr", "s", "127.0.0.1", "FRP 服务器地址")
		cmd.PersistentFlags().IntVarP(&c.ServerPort, "server_port", "P", 7000, "FRP 服务器的端口")
		cmd.PersistentFlags().StringVarP(&c.Transport.Protocol, "protocol", "p", "tcp", fmt.Sprintf("可选值为 %v", validation.SupportedTransportProtocols))
		cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "日志级别")
		cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "控制台或文件路径")
		cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "日志文件最大天数")
		cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "在控制台中禁用日志颜色")
		cmd.PersistentFlags().StringVarP(&c.Transport.TLS.ServerName, "tls_server_name", "", "", "指定 TLS 证书的自定义服务器名称")
		cmd.PersistentFlags().StringVarP(&c.DNSServer, "dns_server", "", "", "指定DNS服务器，而不是使用系统默认服务器")
		c.Transport.TLS.Enable = cmd.PersistentFlags().BoolP("tls_enable", "", true, "启用 FRPC TLS")
	}
	cmd.PersistentFlags().StringVarP(&c.User, "user", "u", "", "用户")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "身份验证令牌")
}

type PortsRangeSliceFlag struct {
	V *[]types.PortsRange
}

func (f *PortsRangeSliceFlag) String() string {
	if f.V == nil {
		return ""
	}
	return types.PortsRangeSlice(*f.V).String()
}

func (f *PortsRangeSliceFlag) Set(s string) error {
	slice, err := types.NewPortsRangeSliceFromString(s)
	if err != nil {
		return err
	}
	*f.V = slice
	return nil
}

func (f *PortsRangeSliceFlag) Type() string {
	return "string"
}

type BoolFuncFlag struct {
	TrueFunc  func()
	FalseFunc func()

	v bool
}

func (f *BoolFuncFlag) String() string {
	return strconv.FormatBool(f.v)
}

func (f *BoolFuncFlag) Set(s string) error {
	f.v = strconv.FormatBool(f.v) == "true"

	if !f.v {
		if f.FalseFunc != nil {
			f.FalseFunc()
		}
		return nil
	}

	if f.TrueFunc != nil {
		f.TrueFunc()
	}
	return nil
}

func (f *BoolFuncFlag) Type() string {
	return "bool"
}

func RegisterServerConfigFlags(cmd *cobra.Command, c *v1.ServerConfig, opts ...RegisterFlagOption) {
	cmd.PersistentFlags().StringVarP(&c.BindAddr, "bind_addr", "", "0.0.0.0", "绑定地址")
	cmd.PersistentFlags().IntVarP(&c.BindPort, "bind_port", "p", 7000, "绑定端口")
	cmd.PersistentFlags().IntVarP(&c.KCPBindPort, "kcp_bind_port", "", 0, "KCP 绑定UDP端口")
	cmd.PersistentFlags().StringVarP(&c.ProxyBindAddr, "proxy_bind_addr", "", "0.0.0.0", "代理绑定地址")
	cmd.PersistentFlags().IntVarP(&c.VhostHTTPPort, "vhost_http_port", "", 0, "虚拟主机HTTP端口")
	cmd.PersistentFlags().IntVarP(&c.VhostHTTPSPort, "vhost_https_port", "", 0, "虚拟主机HTTPS端口")
	cmd.PersistentFlags().Int64VarP(&c.VhostHTTPTimeout, "vhost_http_timeout", "", 60, "虚拟主机HTTP响应标头超时")
	cmd.PersistentFlags().StringVarP(&c.WebServer.Addr, "dashboard_addr", "", "0.0.0.0", "仪表板地址")
	cmd.PersistentFlags().IntVarP(&c.WebServer.Port, "dashboard_port", "", 0, "仪表板端口")
	cmd.PersistentFlags().StringVarP(&c.WebServer.User, "dashboard_user", "", "admin", "仪表板用户")
	cmd.PersistentFlags().StringVarP(&c.WebServer.Password, "dashboard_pwd", "", "admin", "仪表板密码")
	cmd.PersistentFlags().BoolVarP(&c.EnablePrometheus, "enable_prometheus", "", false, "启用Prometheus仪表板")
	cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "日志文件")
	cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "日志级别")
	cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "日志最大天数")
	cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "在控制台中禁用日志颜色")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "身份验证令牌")
	cmd.PersistentFlags().StringVarP(&c.SubDomainHost, "subdomain_host", "", "", "子域名主机")
	cmd.PersistentFlags().VarP(&PortsRangeSliceFlag{V: &c.AllowPorts}, "allow_ports", "", "允许端口")
	cmd.PersistentFlags().Int64VarP(&c.MaxPortsPerClient, "max_ports_per_client", "", 0, "每个客户端的最大端口数")
	cmd.PersistentFlags().BoolVarP(&c.Transport.TLS.Force, "tls_only", "", false, "仅限 FRPS TLS")

	webServerTLS := v1.TLSConfig{}
	cmd.PersistentFlags().StringVarP(&webServerTLS.CertFile, "dashboard_tls_cert_file", "", "", "仪表板TLS证书文件")
	cmd.PersistentFlags().StringVarP(&webServerTLS.KeyFile, "dashboard_tls_key_file", "", "", "仪表板TLS密钥文件")
	cmd.PersistentFlags().VarP(&BoolFuncFlag{
		TrueFunc: func() { c.WebServer.TLS = &webServerTLS },
	}, "dashboard_tls_mode", "", "如果启用仪表板TLS模式")
}
