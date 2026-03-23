package dto

import (
	"github.com/golang-jwt/jwt/v4"
)

type JWTUser struct {
	Environment string  `json:"environment"`
	Data        JWTData `json:"data"`
	jwt.RegisteredClaims
}

// GetEnvironment implements the environmentClaims interface used by JWT
// validation to avoid reflection.
func (u *JWTUser) GetEnvironment() string {
	return u.Environment
}
type JWTData struct {
	UserId     string     `json:"userId"`
	Permission Permission `json:"permission"`
}

type Permission struct {
	ItemManagement RootPermission `json:"ItemManagement"`
}

type RootPermission struct {
	Action []string `json:"action"`
}

func (p *RootPermission) hasAction(action string) bool {
	for _, a := range p.Action {
		if a == action {
			return true
		}
	}
	return false
}

func (p *RootPermission) CanCreateItem() bool {
	return p.hasAction("create") && p.hasAction("read")
}

func (p *RootPermission) CanReadItem() bool {
	return p.hasAction("read")
}

func (p *RootPermission) CanUpdateItem() bool {
	return p.hasAction("update") && p.hasAction("read")
}

func (p *RootPermission) CanDeleteItem() bool {
	return p.hasAction("delete") && p.hasAction("read")
}
