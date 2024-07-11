// Copyright 2023 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sub

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iami317/hepx/pkg/config"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	"github.com/iami317/hepx/pkg/nathole"
)

// nathole discover 是 frp 工具中的一个命令行工具，用于探测当前网络环境下是否存在 NAT 隧道，并尝试通过穿透 NAT 隧道来建立连接。
// 在使用 frp 进行内网穿透时，如果客户端和服务器之间存在 NAT 隧道，就会导致无法建立连接。通过执行 nathole discover 命令，frp 可以探测当前网络环境下是否存在 NAT 隧道，并尝试通过穿透 NAT 隧道来建立连接，从而解决连接问题。
// 在执行 nathole discover 命令时，frp 将会向服务器发送 UDP 数据包，并从服务器的响应中获取 NAT 类型和公网 IP 地址等信息。根据获取到的信息，frp 尝试使用不同的方式来穿透 NAT 隧道，以建立连接。
// 需要注意的是，nathole discover 命令仅适用于 UDP 类型代理，因为它需要使用 UDP 数据包来进行探测。如果您使用的是 TCP 类型代理，则无法使用 nathole discover 命令来穿透 NAT 隧道。
var (
	natHoleSTUNServer string
	natHoleLocalAddr  string
)

func init() {
	rootCmd.AddCommand(natholeCmd)
	natholeCmd.AddCommand(natholeDiscoveryCmd)

	natholeCmd.PersistentFlags().StringVarP(&natHoleSTUNServer, "nat_hole_stun_server", "", "", "nathole 的 STUN 服务器地址")
	natholeCmd.PersistentFlags().StringVarP(&natHoleLocalAddr, "nat_hole_local_addr", "l", "", "连接STUN服务器的本地地址")
}

var natholeCmd = &cobra.Command{
	Use:   "nathole",
	Short: "关于nathole的操作",
}

var natholeDiscoveryCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover nathole information from stun server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ignore error here, because we can use command line pameters
		cfg, _, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
		if err != nil {
			cfg = &v1.ClientCommonConfig{}
			cfg.Complete()
		}
		if natHoleSTUNServer != "" {
			cfg.NatHoleSTUNServer = natHoleSTUNServer
		}

		if err := validateForNatHoleDiscovery(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		addrs, localAddr, err := nathole.Discover([]string{cfg.NatHoleSTUNServer}, natHoleLocalAddr)
		if err != nil {
			fmt.Println("discover error:", err)
			os.Exit(1)
		}
		if len(addrs) < 2 {
			fmt.Printf("discover error: can not get enough addresses, need 2, got: %v\n", addrs)
			os.Exit(1)
		}

		localIPs, _ := nathole.ListLocalIPsForNatHole(10)

		natFeature, err := nathole.ClassifyNATFeature(addrs, localIPs)
		if err != nil {
			fmt.Println("classify nat feature error:", err)
			os.Exit(1)
		}
		fmt.Println("STUN server:", cfg.NatHoleSTUNServer)
		fmt.Println("Your NAT type is:", natFeature.NatType)
		fmt.Println("Behavior is:", natFeature.Behavior)
		fmt.Println("External address is:", addrs)
		fmt.Println("Local address is:", localAddr.String())
		fmt.Println("Public Network:", natFeature.PublicNetwork)
		return nil
	},
}

func validateForNatHoleDiscovery(cfg *v1.ClientCommonConfig) error {
	if cfg.NatHoleSTUNServer == "" {
		return fmt.Errorf("nat_hole_stun_server can not be empty")
	}
	return nil
}
