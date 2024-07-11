package ssh

import (
	"github.com/iami317/hepx/client/proxy"
	v1 "github.com/iami317/hepx/pkg/config/v1"
)

func createSuccessInfo(user string, pc v1.ProxyConfigurer, ps *proxy.WorkingStatus) string {
	base := pc.GetBaseConfig()
	out := "\n"
	out += "frp (via SSH) (Ctrl+C to quit)\n\n"
	out += "User: " + user + "\n"
	out += "ProxyName: " + base.Name + "\n"
	out += "Type: " + base.Type + "\n"
	out += "RemoteAddress: " + ps.RemoteAddr + "\n"
	return out
}
