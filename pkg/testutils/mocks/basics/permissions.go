package basics

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type PermissionsLink struct {
	*chain.Base
	permissions []cfg.Permission
}

func NewPermissionsLink(configs ...cfg.Config) *PermissionsLink {
	pl := &PermissionsLink{}
	pl.Base = chain.NewBase(pl, configs...)
	return pl
}

func (pl *PermissionsLink) WithPermissions(permissions ...cfg.Permission) *PermissionsLink {
	pl.permissions = permissions
	return pl
}

func (pl *PermissionsLink) Permissions() []cfg.Permission {
	return pl.permissions
}

func (pl *PermissionsLink) Process(val any) error {
	pl.Send(val)
	return nil
}
