// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

// Built-in download test servers from various IDC providers
var builtinDownloadServers = []ServerResponse{
	{
		ID:       "url:cloudflare:1gb",
		Name:     "Cloudflare 1GB",
		Host:     "speed.cloudflare.com",
		URL:      "https://speed.cloudflare.com/__down?bytes=1000000000",
		Country:  "Global",
		Sponsor:  "Cloudflare",
		IsPublic: true,
	},
	{
		ID:       "url:ovh:1gb",
		Name:     "OVH FR 1GB",
		Host:     "proof.ovh.net",
		URL:      "https://proof.ovh.net/files/1Gb.dat",
		Country:  "France",
		Sponsor:  "OVH",
		IsPublic: true,
	},
	{
		ID:       "url:linode:eu:1gb",
		Name:     "Linode EU 1GB",
		Host:     "speedtest.frankfurt.linode.com",
		URL:      "http://speedtest.frankfurt.linode.com/1GB-frankfurt.bin",
		Country:  "Germany",
		Sponsor:  "Linode",
		IsPublic: true,
	},
	{
		ID:       "url:linode:us:1gb",
		Name:     "Linode US 1GB",
		Host:     "speedtest.newark.linode.com",
		URL:      "http://speedtest.newark.linode.com/1GB-newark.bin",
		Country:  "United States",
		Sponsor:  "Linode",
		IsPublic: true,
	},
	{
		ID:       "url:linode:jp:1gb",
		Name:     "Linode JP 1GB",
		Host:     "speedtest.tokyo2.linode.com",
		URL:      "http://speedtest.tokyo2.linode.com/1GB-tokyo2.bin",
		Country:  "Japan",
		Sponsor:  "Linode",
		IsPublic: true,
	},
	{
		ID:       "url:vultr:sgp:1gb",
		Name:     "Vultr SG 1GB",
		Host:     "sgp-ping.vultr.com",
		URL:      "https://sgp-ping.vultr.com/vultr.com.1000MB.bin",
		Country:  "Singapore",
		Sponsor:  "Vultr",
		IsPublic: true,
	},
	{
		ID:       "url:datapacket:tyo:1gb",
		Name:     "DataPacket JP 1GB",
		Host:     "tyo.download.datapacket.com",
		URL:      "http://tyo.download.datapacket.com/1000mb.bin",
		Country:  "Japan",
		Sponsor:  "DataPacket",
		IsPublic: true,
	},
	{
		ID:       "url:datapacket:lax:1gb",
		Name:     "DataPacket US-LA 1GB",
		Host:     "lax.download.datapacket.com",
		URL:      "http://lax.download.datapacket.com/1000mb.bin",
		Country:  "United States",
		Sponsor:  "DataPacket",
		IsPublic: true,
	},
	{
		ID:       "url:datapacket:hkg:1gb",
		Name:     "DataPacket HK 1GB",
		Host:     "hkg.download.datapacket.com",
		URL:      "http://hkg.download.datapacket.com/1000mb.bin",
		Country:  "Hong Kong",
		Sponsor:  "DataPacket",
		IsPublic: true,
	},
}
