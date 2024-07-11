package auth

import (
	"github.com/iami317/hepx/pkg/msg"
)

var AlwaysPassVerifier = &alwaysPass{}

var _ Verifier = &alwaysPass{}

type alwaysPass struct{}

func (*alwaysPass) VerifyLogin(*msg.Login) error { return nil }

func (*alwaysPass) VerifyPing(*msg.Ping) error { return nil }

func (*alwaysPass) VerifyNewWorkConn(*msg.NewWorkConn) error { return nil }
