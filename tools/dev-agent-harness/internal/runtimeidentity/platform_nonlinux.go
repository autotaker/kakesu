//go:build !linux

package runtimeidentity

import "errors"

func platformLookupUser(string) (account, error) { return account{}, errors.New("unsupported") }
func platformLookupGroup(string) (group, error)  { return group{}, errors.New("unsupported") }
func platformEUID() int                          { return 0 }
