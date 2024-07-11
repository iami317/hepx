package main

import (
	"context"
	"fmt"
	"gitee.com/menciis/logx"
	_ "github.com/iami317/hepx/assets/frps"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	_ "github.com/iami317/hepx/pkg/metrics"
	"github.com/iami317/hepx/pkg/msg"
	"github.com/iami317/hepx/server"
)

func main() {
	logx.SetLevel("verbose")
	cfg := v1.ServerConfig{
		BindAddr: "0.0.0.0",
		BindPort: 5000,
	}
	cfg.Complete()
	svr, err := server.NewService(&cfg)
	if err != nil {
		logx.Errorf("frps started fail --%v", err)
		return
	}

	svr.OnLoginFn = func(ctx context.Context, loginMsg *msg.NewProxy) {
		fmt.Println("----------------", loginMsg.String())
	}
	logx.Verbosef("frps started successfully")
	svr.Run(context.Background())
	return
}
