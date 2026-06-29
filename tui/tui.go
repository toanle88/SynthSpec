// Package tui re-exports types and functions from the dashboard and welcome subpackages
// so that external consumers (primarily cmd/) can import a single "tui" package.
package tui

import (
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui/dashboard"
	"github.com/toanle/synthspec/tui/welcome"
)

// DashboardModel is a type alias for dashboard.DashboardModel
type DashboardModel = dashboard.DashboardModel

// NewDashboardModel creates a new DashboardModel
func NewDashboardModel(sess *state.Session, gw gateway.Gateway, outputDir string) DashboardModel {
	return dashboard.NewDashboardModel(sess, gw, outputDir)
}

// WelcomeModel is a type alias for welcome.WelcomeModel
type WelcomeModel = welcome.WelcomeModel

// NewWelcomeModel creates a new WelcomeModel
func NewWelcomeModel() WelcomeModel {
	return welcome.NewWelcomeModel()
}

// WelcomeAction is a type alias for welcome.WelcomeAction
type WelcomeAction = welcome.WelcomeAction

// Welcome action constants
const (
	ActionCreate = welcome.ActionCreate
	ActionResume = welcome.ActionResume
	ActionExport = welcome.ActionExport
	ActionExit   = welcome.ActionExit
)
