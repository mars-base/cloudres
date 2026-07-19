package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mars-base/cloudres/internal/core"
)

type providersMsg []*core.Provider

type resourcesMsg struct {
	resources []core.Resource
	err       error
}

func (m *appModel) detectProvidersCmd() tea.Cmd {
	return func() tea.Msg {
		return providersMsg(m.registry.DetectAll(m.ctx))
	}
}

func (m *appModel) fetchResourcesCmd() tea.Cmd {
	prov := m.currentProvider
	rtype := m.currentResource
	region := m.currentRegion
	reg := m.registry
	db := m.db
	ctx := m.ctx

	return func() tea.Msg {
		var fetcher core.ResourceFetcher
		for _, f := range reg.FetchersFor(prov.Name) {
			if f.ResourceType() == rtype {
				fetcher = f
				break
			}
		}
		if fetcher == nil {
			return resourcesMsg{err: &core.ProviderError{
				Provider:  prov.Name,
				Operation: "fetch",
				Stderr:    "no fetcher for " + rtype,
			}}
		}

		resources, err := core.SyncAndList(ctx, db, prov, fetcher, region)
		return resourcesMsg{resources: resources, err: err}
	}
}
