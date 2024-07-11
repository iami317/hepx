package sub

import (
	"fmt"
	"os"
	"strings"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/iami317/hepx/pkg/config"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	clientsdk "github.com/iami317/hepx/pkg/sdk/client"
)

func init() {
	rootCmd.AddCommand(NewAdminCommand(
		"reload",
		"热重载frpc配置",
		ReloadHandler,
	))

	rootCmd.AddCommand(NewAdminCommand(
		"status",
		"所有代理状态概览",
		StatusHandler,
	))

	rootCmd.AddCommand(NewAdminCommand(
		"stop",
		"停止运行的 frpc",
		StopHandler,
	))
}

func NewAdminCommand(name, short string, handler func(*v1.ClientCommonConfig) error) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if cfg.WebServer.Port <= 0 {
				fmt.Println("web server port should be set if you want to use this feature")
				os.Exit(1)
			}

			if err := handler(cfg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
}

func ReloadHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	if err := client.Reload(strictConfigMode); err != nil {
		return err
	}
	fmt.Println("reload success")
	return nil
}

func StatusHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	res, err := client.GetAllProxyStatus()
	if err != nil {
		return err
	}

	fmt.Printf("Proxy Status...\n\n")
	for _, typ := range proxyTypes {
		arrs := res[string(typ)]
		if len(arrs) == 0 {
			continue
		}

		fmt.Println(strings.ToUpper(string(typ)))
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range arrs {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	return nil
}

func StopHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	if err := client.Stop(); err != nil {
		return err
	}
	fmt.Println("stop success")
	return nil
}
