// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package block

import "github.com/juju/1.25-upgrade/juju1/state"

type blockAccess interface {
	AllBlocks() ([]state.Block, error)
	SwitchBlockOn(t state.BlockType, msg string) error
	SwitchBlockOff(t state.BlockType) error
}

type stateShim struct {
	*state.State
}
