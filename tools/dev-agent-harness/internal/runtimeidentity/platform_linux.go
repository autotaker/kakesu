//go:build linux

package runtimeidentity

import (
	"os"
	"os/user"
)

func platformLookupUser(name string) (account, error) {
	u, err := user.Lookup(name)
	if err != nil || u == nil {
		return account{}, ErrDenied
	}
	return account{UID: u.Uid, GID: u.Gid}, nil
}

func platformLookupGroup(name string) (group, error) {
	g, err := user.LookupGroup(name)
	if err != nil || g == nil {
		return group{}, ErrDenied
	}
	return group{GID: g.Gid}, nil
}

func platformEUID() int { return os.Geteuid() }
