// Copyright 2018 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"gitee.com/menciis/logx"
	"github.com/iami317/hepx/pkg/config"
	"github.com/iami317/hepx/pkg/util/system"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/iami317/hepx/assets/frps"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	"github.com/iami317/hepx/pkg/config/v1/validation"
	_ "github.com/iami317/hepx/pkg/metrics"
	"github.com/iami317/hepx/server"
)

var (
	cfgFile          string
	strictConfigMode bool

	serverCfg v1.ServerConfig
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "FRPS的配置文件")
	rootCmd.PersistentFlags().BoolVarP(&strictConfigMode, "strict_config", "", true, "严格的配置解析模式，未知字段会导致错误")

	config.RegisterServerConfigFlags(rootCmd, &serverCfg)
}

var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps is the server of frp (https://github.com/iami317/hepx)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			svrCfg         *v1.ServerConfig
			isLegacyFormat bool
			err            error
		)
		if cfgFile != "" {
			svrCfg, isLegacyFormat, err = config.LoadServerConfig(cfgFile, strictConfigMode)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if isLegacyFormat {
				fmt.Printf("警告：ini 格式已弃用，将来将删除支持，请改用 yaml/json/toml 格式！\n")
			}
		} else {
			serverCfg.Complete()
			svrCfg = &serverCfg
		}

		warning, err := validation.ValidateServerConfig(svrCfg)
		if warning != nil {
			fmt.Printf("WARNING: %v\n", warning)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := runServer(svrCfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

func main() {
	system.EnableCompatibilityMode()
	Execute()
}

func Execute() {
	rootCmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cfg *v1.ServerConfig) (err error) {
	if cfgFile != "" {
		logx.Verbosef("frps uses config file: %s", cfgFile)
	} else {
		logx.Verbosef("frps uses command line arguments for config")
	}
	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	logx.Verbosef("frps started successfully")
	svr.Run(context.Background())
	return
}
