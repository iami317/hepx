package main

import (
	"context"
	"fmt"
	"gitee.com/menciis/logx"
	_ "github.com/iami317/hepx/assets/frpc"
	"github.com/iami317/hepx/client"
	"github.com/iami317/hepx/cmd/frpc/sub"
	"github.com/iami317/hepx/pkg/config"
	v1 "github.com/iami317/hepx/pkg/config/v1"
)

func main() {
	logx.SetLevel("verbose")

	cfg := v1.ClientConfig{
		ClientCommonConfig: v1.ClientCommonConfig{
			Auth: v1.AuthClientConfig{
				Method: v1.AuthMethodToken,
				Token:  "admin",
			},
			ServerAddr: "192.168.8.109",
			ServerPort: 5000,
			Metadatas:  map[string]string{"attack_host": "192.168.8.114", "ips": "192.168.1.141"},
		},
		Proxies: []v1.TypedProxyConfig{
			{
				ProxyConfigurer: &v1.TCPProxyConfig{
					//RemotePort: 1085,
					ProxyBaseConfig: v1.ProxyBaseConfig{
						Type: "tcp",
						Name: "socks5_proxy",
						Transport: v1.ProxyTransport{
							UseEncryption:  true,
							UseCompression: true,
						},
						ProxyBackend: v1.ProxyBackend{
							Plugin: v1.TypedClientPluginOptions{
								ClientPluginOptions: v1.Socks5PluginOptions{
									Type:     "socks5",
									Username: "abc",
									Password: "abc",
								},
							},
						},
					},
				},
			},
		},
	}
	commonCfg, proxyCfg, visitorCfg := config.LoadConfig(&cfg)
	options := client.ServiceOptions{
		Common:      commonCfg,
		ProxyCfgs:   proxyCfg,
		VisitorCfgs: visitorCfg,
	}
	svr, err := client.NewService(options)
	if err != nil {
		return
	}

	shouldGracefulClose := commonCfg.Transport.Protocol == "kcp" || commonCfg.Transport.Protocol == "quic"
	if shouldGracefulClose {
		go sub.HandleTermSignal(svr)
	}
	err = svr.Run(context.Background())
	if err != nil {
		fmt.Println("---------", err)
	}
	return
}
