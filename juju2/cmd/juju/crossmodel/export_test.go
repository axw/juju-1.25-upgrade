// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel

import (
	"github.com/juju/cmd"

	"github.com/juju/1.25-upgrade/juju2/cmd/modelcmd"
	"github.com/juju/1.25-upgrade/juju2/jujuclient"
)

var (
	Max          = max
	DescAt       = descAt
	BreakLines   = breakLines
	ColumnWidth  = columnWidth
	BreakOneWord = breakOneWord
)

func noOpRefresh(jujuclient.ClientStore, string) error {
	return nil
}

func NewOfferCommandForTest(
	store jujuclient.ClientStore,
	api OfferAPI,
) cmd.Command {
	aCmd := &offerCommand{
		newAPIFunc: func() (OfferAPI, error) {
			return api, nil
		},
		refreshModels: noOpRefresh,
	}
	aCmd.SetClientStore(store)
	return modelcmd.WrapController(aCmd)
}

func NewShowEndpointsCommandForTest(store jujuclient.ClientStore, api ShowAPI) cmd.Command {
	aCmd := &showCommand{newAPIFunc: func() (ShowAPI, error) {
		return api, nil
	}}
	aCmd.SetClientStore(store)
	return modelcmd.WrapController(aCmd)
}

func NewListEndpointsCommandForTest(store jujuclient.ClientStore, api ListAPI) cmd.Command {
	aCmd := &listCommand{newAPIFunc: func() (ListAPI, error) {
		return api, nil
	}}
	aCmd.SetClientStore(store)
	return modelcmd.WrapController(aCmd)
}

func NewFindEndpointsCommandForTest(store jujuclient.ClientStore, api FindAPI) cmd.Command {
	aCmd := &findCommand{newAPIFunc: func() (FindAPI, error) {
		return api, nil
	}}
	aCmd.SetClientStore(store)
	return modelcmd.WrapController(aCmd)
}
