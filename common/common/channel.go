package common

import "github.com/ddr4869/minifab/common/configtx"

type Channel interface {
	GetOrganization(orgName string) *configtx.Organization
	GetChannelConfig() configtx.ConfigTx
}

type channel struct {
	Name          string
	Organizations map[string]configtx.Organization
	ChannelConfig configtx.ConfigTx
}

func NewChannel(name string, channelConfig configtx.ConfigTx) Channel {
	channelOrganizations := make(map[string]configtx.Organization)

	for _, organization := range channelConfig.Organizations {
		channelOrganizations[organization.Name] = organization
	}

	return &channel{
		Name:          name,
		Organizations: channelOrganizations,
		ChannelConfig: channelConfig,
	}
}

func (c *channel) GetName() string {
	return c.Name
}

func (c *channel) GetOrganization(orgName string) *configtx.Organization {
	org, ok := c.Organizations[orgName]
	if !ok {
		return nil
	}
	return &org
}

func (c *channel) GetChannelConfig() configtx.ConfigTx {
	return c.ChannelConfig
}
