package cfg

import "fmt"

type Permission struct {
	Platform   Platform
	Permission string
}

func (p Permission) String() string {
	return fmt.Sprintf("%v:%s", p.Platform, p.Permission)
}

func NewPermission(platform Platform, permission string) Permission {
	return Permission{Platform: platform, Permission: permission}
}
