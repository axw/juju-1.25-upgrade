// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package juju_test

import (
	"fmt"
	stdtesting "testing"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/charmrepo.v2-unstable"

	"github.com/juju/1.25-upgrade/juju2/constraints"
	"github.com/juju/1.25-upgrade/juju2/instance"
	"github.com/juju/1.25-upgrade/juju2/juju"
	"github.com/juju/1.25-upgrade/juju2/juju/testing"
	"github.com/juju/1.25-upgrade/juju2/state"
	"github.com/juju/1.25-upgrade/juju2/testcharms"
	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

func Test(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

// DeployLocalSuite uses a fresh copy of the same local dummy charm for each
// test, because DeployApplication demands that a charm already exists in state,
// and that's is the simplest way to get one in there.
type DeployLocalSuite struct {
	testing.JujuConnSuite
	repo  charmrepo.Interface
	charm *state.Charm
}

var _ = gc.Suite(&DeployLocalSuite{})

func (s *DeployLocalSuite) SetUpSuite(c *gc.C) {
	s.JujuConnSuite.SetUpSuite(c)
	s.repo = &charmrepo.LocalRepository{Path: testcharms.Repo.Path()}
	s.PatchValue(&charmrepo.CacheDir, c.MkDir())
}

func (s *DeployLocalSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	curl := charm.MustParseURL("local:quantal/dummy")
	charm, err := testing.PutCharm(s.State, curl, s.repo, false)
	c.Assert(err, jc.ErrorIsNil)
	s.charm = charm
}

func (s *DeployLocalSuite) TestDeployMinimal(c *gc.C) {
	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
		})
	c.Assert(err, jc.ErrorIsNil)
	s.assertCharm(c, app, s.charm.URL())
	s.assertSettings(c, app, charm.Settings{})
	s.assertConstraints(c, app, constraints.Value{})
	s.assertMachines(c, app, constraints.Value{})
}

func (s *DeployLocalSuite) TestDeploySeries(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Series:          "aseries",
		})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Series, gc.Equals, "aseries")
}

func (s *DeployLocalSuite) TestDeployWithImplicitBindings(c *gc.C) {
	wordpressCharm := s.addWordpressCharmWithExtraBindings(c)

	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName:  "bob",
			Charm:            wordpressCharm,
			EndpointBindings: nil,
		})
	c.Assert(err, jc.ErrorIsNil)

	s.assertBindings(c, app, map[string]string{
		// relation names
		"url":             "",
		"logging-dir":     "",
		"monitoring-port": "",
		"db":              "",
		"cache":           "",
		"cluster":         "",
		// extra-bindings names
		"db-client": "",
		"admin-api": "",
		"foo-bar":   "",
	})
}

func (s *DeployLocalSuite) addWordpressCharm(c *gc.C) *state.Charm {
	wordpressCharmURL := charm.MustParseURL("local:quantal/wordpress")
	return s.addWordpressCharmFromURL(c, wordpressCharmURL)
}

func (s *DeployLocalSuite) addWordpressCharmWithExtraBindings(c *gc.C) *state.Charm {
	wordpressCharmURL := charm.MustParseURL("local:quantal/wordpress-extra-bindings")
	return s.addWordpressCharmFromURL(c, wordpressCharmURL)
}

func (s *DeployLocalSuite) addWordpressCharmFromURL(c *gc.C, charmURL *charm.URL) *state.Charm {
	wordpressCharm, err := testing.PutCharm(s.State, charmURL, s.repo, false)
	c.Assert(err, jc.ErrorIsNil)
	return wordpressCharm
}

func (s *DeployLocalSuite) assertBindings(c *gc.C, app *state.Application, expected map[string]string) {
	bindings, err := app.EndpointBindings()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(bindings, jc.DeepEquals, expected)
}

func (s *DeployLocalSuite) TestDeployWithSomeSpecifiedBindings(c *gc.C) {
	wordpressCharm := s.addWordpressCharm(c)
	_, err := s.State.AddSpace("db", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("public", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)

	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           wordpressCharm,
			EndpointBindings: map[string]string{
				"":   "public",
				"db": "db",
			},
		})
	c.Assert(err, jc.ErrorIsNil)

	s.assertBindings(c, app, map[string]string{
		// default binding
		"": "public",
		// relation names
		"url":             "public",
		"logging-dir":     "public",
		"monitoring-port": "public",
		"db":              "db",
		"cache":           "public",
		// extra-bindings names
		"db-client": "public",
		"admin-api": "public",
		"foo-bar":   "public",
	})
}

func (s *DeployLocalSuite) TestDeployWithBoundRelationNamesAndExtraBindingsNames(c *gc.C) {
	wordpressCharm := s.addWordpressCharmWithExtraBindings(c)
	_, err := s.State.AddSpace("db", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("public", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("internal", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)

	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           wordpressCharm,
			EndpointBindings: map[string]string{
				"":          "public",
				"db":        "db",
				"db-client": "db",
				"admin-api": "internal",
			},
		})
	c.Assert(err, jc.ErrorIsNil)

	s.assertBindings(c, app, map[string]string{
		"":                "public",
		"url":             "public",
		"logging-dir":     "public",
		"monitoring-port": "public",
		"db":              "db",
		"cache":           "public",
		"db-client":       "db",
		"admin-api":       "internal",
		"cluster":         "public",
		"foo-bar":         "public", // like for relations, uses the application-default.
	})

}

func (s *DeployLocalSuite) TestDeployWithInvalidSpace(c *gc.C) {
	wordpressCharm := s.addWordpressCharm(c)
	_, err := s.State.AddSpace("db", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("public", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("internal", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)

	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           wordpressCharm,
			EndpointBindings: map[string]string{
				"":   "public",
				"db": "intranel", //typo 'intranel'
			},
		})
	c.Assert(err, gc.ErrorMatches, `cannot add application "bob": unknown space "intranel" not valid`)
	c.Check(app, gc.IsNil)
	// The application should not have been added
	_, err = s.State.Application("bob")
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

}

func (s *DeployLocalSuite) TestDeployWithInvalidBinding(c *gc.C) {
	wordpressCharm := s.addWordpressCharm(c)
	_, err := s.State.AddSpace("db", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("public", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.State.AddSpace("internal", "", nil, false)
	c.Assert(err, jc.ErrorIsNil)

	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           wordpressCharm,
			EndpointBindings: map[string]string{
				"":      "public",
				"bd":    "internal", // typo 'bd'
				"pubic": "public",   // typo 'pubic'
			},
		})
	c.Assert(err, gc.ErrorMatches,
		`invalid binding\(s\) supplied "bd", "pubic", valid binding names are "admin-api", .* "url"`)
	c.Check(app, gc.IsNil)
	// The application should not have been added
	_, err = s.State.Application("bob")
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

}

func (s *DeployLocalSuite) TestDeployResources(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			EndpointBindings: map[string]string{
				"": "public",
			},
			Resources: map[string]string{"foo": "bar"},
		})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Resources, gc.DeepEquals, map[string]string{"foo": "bar"})
}

func (s *DeployLocalSuite) TestDeploySettings(c *gc.C) {
	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			ConfigSettings: charm.Settings{
				"title":       "banana cupcakes",
				"skill-level": 9901,
			},
		})
	c.Assert(err, jc.ErrorIsNil)
	s.assertSettings(c, app, charm.Settings{
		"title":       "banana cupcakes",
		"skill-level": int64(9901),
	})
}

func (s *DeployLocalSuite) TestDeploySettingsError(c *gc.C) {
	_, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			ConfigSettings: charm.Settings{
				"skill-level": 99.01,
			},
		})
	c.Assert(err, gc.ErrorMatches, `option "skill-level" expected int, got 99.01`)
	_, err = s.State.Application("bob")
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *DeployLocalSuite) TestDeployConstraints(c *gc.C) {
	err := s.State.SetModelConstraints(constraints.MustParse("mem=2G"))
	c.Assert(err, jc.ErrorIsNil)
	serviceCons := constraints.MustParse("cores=2")
	app, err := juju.DeployApplication(s.State,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
		})
	c.Assert(err, jc.ErrorIsNil)
	s.assertConstraints(c, app, serviceCons)
}

func (s *DeployLocalSuite) TestDeployNumUnits(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	serviceCons := constraints.MustParse("cores=2")
	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
			NumUnits:        2,
		})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Constraints, gc.DeepEquals, serviceCons)
	c.Assert(f.args.NumUnits, gc.Equals, 2)
}

func (s *DeployLocalSuite) TestDeployForceMachineId(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	serviceCons := constraints.MustParse("cores=2")
	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
			NumUnits:        1,
			Placement:       []*instance.Placement{instance.MustParsePlacement("0")},
		})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Constraints, gc.DeepEquals, serviceCons)
	c.Assert(f.args.NumUnits, gc.Equals, 1)
	c.Assert(f.args.Placement, gc.HasLen, 1)
	c.Assert(*f.args.Placement[0], gc.Equals, instance.Placement{Scope: instance.MachineScope, Directive: "0"})
}

func (s *DeployLocalSuite) TestDeployForceMachineIdWithContainer(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	serviceCons := constraints.MustParse("cores=2")
	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
			NumUnits:        1,
			Placement:       []*instance.Placement{instance.MustParsePlacement(fmt.Sprintf("%s:0", instance.LXD))},
		})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Constraints, gc.DeepEquals, serviceCons)
	c.Assert(f.args.NumUnits, gc.Equals, 1)
	c.Assert(f.args.Placement, gc.HasLen, 1)
	c.Assert(*f.args.Placement[0], gc.Equals, instance.Placement{Scope: string(instance.LXD), Directive: "0"})
}

func (s *DeployLocalSuite) TestDeploy(c *gc.C) {
	f := &fakeDeployer{State: s.State}

	serviceCons := constraints.MustParse("cores=2")
	placement := []*instance.Placement{
		{Scope: s.State.ModelUUID(), Directive: "valid"},
		{Scope: "#", Directive: "0"},
		{Scope: "lxd", Directive: "1"},
		{Scope: "lxd", Directive: ""},
	}
	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
			NumUnits:        4,
			Placement:       placement,
		})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Constraints, gc.DeepEquals, serviceCons)
	c.Assert(f.args.NumUnits, gc.Equals, 4)
	c.Assert(f.args.Placement, gc.DeepEquals, placement)
}

func (s *DeployLocalSuite) TestDeployWithFewerPlacement(c *gc.C) {
	f := &fakeDeployer{State: s.State}
	serviceCons := constraints.MustParse("cores=2")
	placement := []*instance.Placement{{Scope: s.State.ModelUUID(), Directive: "valid"}}
	_, err := juju.DeployApplication(f,
		juju.DeployApplicationParams{
			ApplicationName: "bob",
			Charm:           s.charm,
			Constraints:     serviceCons,
			NumUnits:        3,
			Placement:       placement,
		})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(f.args.Name, gc.Equals, "bob")
	c.Assert(f.args.Charm, gc.DeepEquals, s.charm)
	c.Assert(f.args.Constraints, gc.DeepEquals, serviceCons)
	c.Assert(f.args.NumUnits, gc.Equals, 3)
	c.Assert(f.args.Placement, gc.DeepEquals, placement)
}

func (s *DeployLocalSuite) assertCharm(c *gc.C, app *state.Application, expect *charm.URL) {
	curl, force := app.CharmURL()
	c.Assert(curl, gc.DeepEquals, expect)
	c.Assert(force, jc.IsFalse)
}

func (s *DeployLocalSuite) assertSettings(c *gc.C, app *state.Application, expect charm.Settings) {
	settings, err := app.ConfigSettings()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(settings, gc.DeepEquals, expect)
}

func (s *DeployLocalSuite) assertConstraints(c *gc.C, app *state.Application, expect constraints.Value) {
	cons, err := app.Constraints()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cons, gc.DeepEquals, expect)
}

func (s *DeployLocalSuite) assertMachines(c *gc.C, app *state.Application, expectCons constraints.Value, expectIds ...string) {
	units, err := app.AllUnits()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(units, gc.HasLen, len(expectIds))
	// first manually tell state to assign all the units
	for _, unit := range units {
		id := unit.Tag().Id()
		res, err := s.State.AssignStagedUnits([]string{id})
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(res[0].Error, jc.ErrorIsNil)
		c.Assert(res[0].Unit, gc.Equals, id)
	}

	// refresh the list of units from state
	units, err = app.AllUnits()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(units, gc.HasLen, len(expectIds))
	unseenIds := set.NewStrings(expectIds...)
	for _, unit := range units {
		id, err := unit.AssignedMachineId()
		c.Assert(err, jc.ErrorIsNil)
		unseenIds.Remove(id)
		machine, err := s.State.Machine(id)
		c.Assert(err, jc.ErrorIsNil)
		cons, err := machine.Constraints()
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(cons, gc.DeepEquals, expectCons)
	}
	c.Assert(unseenIds, gc.DeepEquals, set.NewStrings())
}

type fakeDeployer struct {
	*state.State
	args state.AddApplicationArgs
}

func (f *fakeDeployer) AddApplication(args state.AddApplicationArgs) (*state.Application, error) {
	f.args = args
	return nil, nil
}
