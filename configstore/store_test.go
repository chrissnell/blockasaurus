// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package configstore

import (
	"os"
	"path/filepath"

	"github.com/0xERR0R/blocky/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("ConfigStore", func() {
	var store *ConfigStore

	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		var err error
		store, err = Open(dbPath)
		Expect(err).Should(Succeed())
		DeferCleanup(store.Close)
	})

	Describe("ClientGroup CRUD", func() {
		It("should create and list client groups", func() {
			g := &ClientGroup{
				Name:    "kids",
				Clients: StringList{"192.168.1.0/24", "kid-laptop"},
				Groups:  StringList{"ads", "malware"},
			}
			Expect(store.PutClientGroup(g)).Should(Succeed())

			groups, err := store.ListClientGroups()
			Expect(err).Should(Succeed())
			Expect(groups).Should(HaveLen(1))
			Expect(groups[0].Name).Should(Equal("kids"))
			Expect([]string(groups[0].Clients)).Should(ConsistOf("192.168.1.0/24", "kid-laptop"))
			Expect([]string(groups[0].Groups)).Should(ConsistOf("ads", "malware"))
		})

		It("should upsert by name", func() {
			g := &ClientGroup{
				Name:    "kids",
				Clients: StringList{"10.0.0.1"},
				Groups:  StringList{"ads"},
			}
			Expect(store.PutClientGroup(g)).Should(Succeed())
			originalID := g.ID

			g2 := &ClientGroup{
				Name:    "kids",
				Clients: StringList{"10.0.0.2"},
				Groups:  StringList{"ads", "malware"},
			}
			Expect(store.PutClientGroup(g2)).Should(Succeed())

			groups, err := store.ListClientGroups()
			Expect(err).Should(Succeed())
			Expect(groups).Should(HaveLen(1))
			Expect(groups[0].ID).Should(Equal(originalID))
			Expect([]string(groups[0].Clients)).Should(ConsistOf("10.0.0.2"))
		})

		It("should get by name", func() {
			Expect(store.PutClientGroup(&ClientGroup{
				Name:    "office",
				Clients: StringList{"10.0.0.0/8"},
				Groups:  StringList{"tracking"},
			})).Should(Succeed())

			g, err := store.GetClientGroup("office")
			Expect(err).Should(Succeed())
			Expect(g.Name).Should(Equal("office"))
		})

		It("should return error for missing group", func() {
			_, err := store.GetClientGroup("nonexistent")
			Expect(err).Should(HaveOccurred())
		})

		It("should delete by name", func() {
			Expect(store.PutClientGroup(&ClientGroup{
				Name:    "temp",
				Clients: StringList{},
				Groups:  StringList{},
			})).Should(Succeed())

			Expect(store.DeleteClientGroup("temp")).Should(Succeed())

			groups, err := store.ListClientGroups()
			Expect(err).Should(Succeed())
			Expect(groups).Should(BeEmpty())
		})

		It("should return error deleting nonexistent group", func() {
			err := store.DeleteClientGroup("ghost")
			Expect(err).Should(MatchError(gorm.ErrRecordNotFound))
		})
	})

	Describe("BlocklistSource CRUD", func() {
		It("should create and list sources", func() {
			src := &BlocklistSource{
				GroupName:  "ads",
				ListType:   "deny",
				SourceType: "http",
				Source:     "https://example.com/ads.txt",
				Enabled:    new(true),
			}
			Expect(store.CreateBlocklistSource(src)).Should(Succeed())
			Expect(src.ID).ShouldNot(BeZero())

			sources, err := store.ListBlocklistSources("", "")
			Expect(err).Should(Succeed())
			Expect(sources).Should(HaveLen(1))
		})

		It("should filter by group and type", func() {
			Expect(store.CreateBlocklistSource(&BlocklistSource{
				GroupName: "ads", ListType: "deny", SourceType: "http", Source: "https://a.com", Enabled: new(true),
			})).Should(Succeed())
			Expect(store.CreateBlocklistSource(&BlocklistSource{
				GroupName: "malware", ListType: "deny", SourceType: "http", Source: "https://b.com", Enabled: new(true),
			})).Should(Succeed())
			Expect(store.CreateBlocklistSource(&BlocklistSource{
				GroupName: "ads", ListType: "allow", SourceType: "http", Source: "https://c.com", Enabled: new(true),
			})).Should(Succeed())

			byGroup, err := store.ListBlocklistSources("ads", "")
			Expect(err).Should(Succeed())
			Expect(byGroup).Should(HaveLen(2))

			byType, err := store.ListBlocklistSources("", "deny")
			Expect(err).Should(Succeed())
			Expect(byType).Should(HaveLen(2))

			byBoth, err := store.ListBlocklistSources("ads", "deny")
			Expect(err).Should(Succeed())
			Expect(byBoth).Should(HaveLen(1))
		})

		It("should update a source", func() {
			src := &BlocklistSource{
				GroupName: "ads", ListType: "deny", SourceType: "http", Source: "https://old.com", Enabled: new(true),
			}
			Expect(store.CreateBlocklistSource(src)).Should(Succeed())

			src.Source = "https://new.com"
			Expect(store.UpdateBlocklistSource(src)).Should(Succeed())

			got, err := store.GetBlocklistSource(src.ID)
			Expect(err).Should(Succeed())
			Expect(got.Source).Should(Equal("https://new.com"))
		})

		It("should delete a source", func() {
			src := &BlocklistSource{
				GroupName: "ads", ListType: "deny", SourceType: "http", Source: "https://x.com", Enabled: new(true),
			}
			Expect(store.CreateBlocklistSource(src)).Should(Succeed())

			Expect(store.DeleteBlocklistSource(src.ID)).Should(Succeed())

			_, err := store.GetBlocklistSource(src.ID)
			Expect(err).Should(HaveOccurred())
		})

		It("should return error deleting nonexistent source", func() {
			err := store.DeleteBlocklistSource(9999)
			Expect(err).Should(MatchError(gorm.ErrRecordNotFound))
		})
	})

	Describe("CustomDNSEntry CRUD", func() {
		It("should create and list entries", func() {
			e := &CustomDNSEntry{
				Domain: "example.com", RecordType: "A", Value: "1.2.3.4", TTL: 3600, Enabled: new(true),
			}
			Expect(store.CreateCustomDNSEntry(e)).Should(Succeed())

			entries, err := store.ListCustomDNSEntries()
			Expect(err).Should(Succeed())
			Expect(entries).Should(HaveLen(1))
			Expect(entries[0].Domain).Should(Equal("example.com"))
		})

		It("should enforce unique constraint on domain+type+value", func() {
			e1 := &CustomDNSEntry{
				Domain: "example.com", RecordType: "A", Value: "1.2.3.4", TTL: 3600, Enabled: new(true),
			}
			Expect(store.CreateCustomDNSEntry(e1)).Should(Succeed())

			e2 := &CustomDNSEntry{
				Domain: "example.com", RecordType: "A", Value: "1.2.3.4", TTL: 7200, Enabled: new(true),
			}
			err := store.CreateCustomDNSEntry(e2)
			Expect(err).Should(HaveOccurred())
		})

		It("should allow same domain with different record types", func() {
			Expect(store.CreateCustomDNSEntry(&CustomDNSEntry{
				Domain: "example.com", RecordType: "A", Value: "1.2.3.4", TTL: 3600, Enabled: new(true),
			})).Should(Succeed())
			Expect(store.CreateCustomDNSEntry(&CustomDNSEntry{
				Domain: "example.com", RecordType: "AAAA", Value: "::1", TTL: 3600, Enabled: new(true),
			})).Should(Succeed())

			entries, err := store.ListCustomDNSEntries()
			Expect(err).Should(Succeed())
			Expect(entries).Should(HaveLen(2))
		})

		It("should delete an entry", func() {
			e := &CustomDNSEntry{
				Domain: "del.com", RecordType: "A", Value: "5.6.7.8", TTL: 300, Enabled: new(true),
			}
			Expect(store.CreateCustomDNSEntry(e)).Should(Succeed())
			Expect(store.DeleteCustomDNSEntry(e.ID)).Should(Succeed())

			entries, err := store.ListCustomDNSEntries()
			Expect(err).Should(Succeed())
			Expect(entries).Should(BeEmpty())
		})
	})

	Describe("RebindingSettings", func() {
		It("should return defaults on first access", func() {
			rs, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())
			Expect(rs.Enabled).Should(BeFalse())
			Expect(rs.AllowedDomains).Should(BeEmpty())
		})

		It("should round-trip the allowlist", func() {
			rs := &RebindingSettings{Enabled: true, AllowedDomains: StringList{"nas.example.com"}}
			Expect(store.PutRebindingSettings(rs)).Should(Succeed())

			got, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())
			Expect(got.Enabled).Should(BeTrue())
			Expect([]string(got.AllowedDomains)).Should(Equal([]string{"nas.example.com"}))
		})

		It("should persist disabling, not skip the zero value", func() {
			Expect(store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"nas.example.com"},
			})).Should(Succeed())

			// gorm's Save skips zero-valued struct fields, which would leave the
			// protection stuck on after a UI toggle-off
			Expect(store.PutRebindingSettings(&RebindingSettings{Enabled: false})).Should(Succeed())

			got, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())
			Expect(got.Enabled).Should(BeFalse())
			Expect(got.AllowedDomains).Should(BeEmpty())
		})

		It("should reject a wildcard entry", func() {
			err := store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"*.example.com"},
			})
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("plain domain"))
		})

		It("should not create a row as a side effect of reading", func() {
			_, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())

			// a read that created an empty row would make the YAML seed a no-op
			// and silently drop an upgrading operator's allowlist
			var count int64
			Expect(store.db.Model(&RebindingSettings{}).Count(&count).Error).Should(Succeed())
			Expect(count).Should(BeZero())
		})

		It("should write through even if the row was never seeded", func() {
			Expect(store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"nas.example.com"},
			})).Should(Succeed())

			got, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())
			Expect(got.Enabled).Should(BeTrue())
		})
	})

	Describe("BuildRebindingConfig", func() {
		It("should seed the singleton from YAML on first run", func() {
			base := config.RebindingProtection{Enable: true, AllowedDomains: []string{"intranet.example.com"}}

			got, err := store.BuildRebindingConfig(base)
			Expect(err).Should(Succeed())
			Expect(got.Enable).Should(BeTrue())
			Expect(got.AllowedDomains).Should(Equal([]string{"intranet.example.com"}))

			// the seed must land in the DB, or the UI would show an empty list
			// while the resolver was enforcing the YAML one
			rs, err := store.GetRebindingSettings()
			Expect(err).Should(Succeed())
			Expect([]string(rs.AllowedDomains)).Should(Equal([]string{"intranet.example.com"}))
		})

		It("should not clobber an explicit setting when re-seeded", func() {
			// Reconfigure calls BuildRebindingConfig on every Apply; a seed that
			// ran more than once would overwrite the user's edit with the YAML
			base := config.RebindingProtection{Enable: false, AllowedDomains: []string{"yaml.example.com"}}
			_, err := store.BuildRebindingConfig(base)
			Expect(err).Should(Succeed())

			Expect(store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"ui.example.com"},
			})).Should(Succeed())

			got, err := store.BuildRebindingConfig(base)
			Expect(err).Should(Succeed())
			Expect(got.Enable).Should(BeTrue())
			Expect(got.AllowedDomains).Should(Equal([]string{"ui.example.com"}))
		})

		It("should let DB state win once seeded", func() {
			Expect(store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"nas.example.com"},
			})).Should(Succeed())

			base := config.RebindingProtection{Enable: false, AllowedDomains: []string{"stale.example.com"}}

			got, err := store.BuildRebindingConfig(base)
			Expect(err).Should(Succeed())
			Expect(got.Enable).Should(BeTrue())
			Expect(got.AllowedDomains).Should(Equal([]string{"nas.example.com"}))
		})

		It("should hand the resolver the DB allowlist, normalized", func() {
			Expect(store.PutRebindingSettings(&RebindingSettings{
				Enabled: true, AllowedDomains: StringList{"NAS.Example.COM."},
			})).Should(Succeed())

			got, err := store.BuildRebindingConfig(config.RebindingProtection{})
			Expect(err).Should(Succeed())
			Expect(got.NormalizedAllowedDomains()).Should(Equal([]string{"nas.example.com"}))
		})
	})

	Describe("BlockSettings", func() {
		It("should return defaults on first access", func() {
			bs, err := store.GetBlockSettings()
			Expect(err).Should(Succeed())
			Expect(bs.BlockType).Should(Equal("ZEROIP"))
			Expect(bs.BlockTTL).Should(Equal("6h"))
		})

		It("should update settings", func() {
			bs := &BlockSettings{BlockType: "NXDOMAIN", BlockTTL: "1h"}
			Expect(store.PutBlockSettings(bs)).Should(Succeed())

			got, err := store.GetBlockSettings()
			Expect(err).Should(Succeed())
			Expect(got.BlockType).Should(Equal("NXDOMAIN"))
			Expect(got.BlockTTL).Should(Equal("1h"))
		})

		It("should reject invalid TTL", func() {
			bs := &BlockSettings{BlockType: "ZEROIP", BlockTTL: "not-a-duration"}
			err := store.PutBlockSettings(bs)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("invalid block TTL"))
		})
	})

	Describe("Open", func() {
		It("should create DB file if it doesn't exist", func() {
			tmpDir := GinkgoT().TempDir()
			dbPath := filepath.Join(tmpDir, "new.db")

			s, err := Open(dbPath)
			Expect(err).Should(Succeed())
			defer s.Close()

			_, err = os.Stat(dbPath)
			Expect(err).Should(Succeed())
		})
	})
})
