// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package configapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"

	"github.com/0xERR0R/blocky/api/configapi"
	"github.com/0xERR0R/blocky/configstore"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockReconfigurer struct {
	err   error
	calls int
}

func (m *mockReconfigurer) Reconfigure(_ context.Context) error {
	m.calls++

	return m.err
}

var _ = Describe("ConfigAPI Handler", func() {
	var (
		store  *configstore.ConfigStore
		reconf *mockReconfigurer
		h      *configapi.ConfigHandler
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		store, err = configstore.Open(filepath.Join(GinkgoT().TempDir(), "test.db"))
		Expect(err).Should(Succeed())
		DeferCleanup(store.Close)

		reconf = &mockReconfigurer{}
		h = configapi.NewConfigHandler(store, reconf, configapi.DoH3Runtime{}, configapi.QueryLogHistorySource{})
	})

	// --- Client Groups ---

	Describe("ClientGroups", func() {
		It("should list empty", func() {
			resp, err := h.ListClientGroups(ctx, configapi.ListClientGroupsRequestObject{})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.ListClientGroups200JSONResponse{}))
			Expect(resp.(configapi.ListClientGroups200JSONResponse)).Should(BeEmpty())
		})

		It("should put and get", func() {
			putResp, err := h.PutClientGroup(ctx, configapi.PutClientGroupRequestObject{
				Name: "kids",
				Body: &configapi.ClientGroupInput{
					Clients: &[]string{"192.168.1.0/24"},
					Groups:  &[]string{"ads"},
				},
			})
			Expect(err).Should(Succeed())
			Expect(putResp).Should(BeAssignableToTypeOf(configapi.PutClientGroup200JSONResponse{}))
			cg := putResp.(configapi.PutClientGroup200JSONResponse)
			Expect(cg.Name).Should(Equal("kids"))
			Expect(cg.Clients).Should(ConsistOf("192.168.1.0/24"))

			getResp, err := h.GetClientGroup(ctx, configapi.GetClientGroupRequestObject{Name: "kids"})
			Expect(err).Should(Succeed())
			Expect(getResp).Should(BeAssignableToTypeOf(configapi.GetClientGroup200JSONResponse{}))
		})

		It("should return 404 for missing group", func() {
			resp, err := h.GetClientGroup(ctx, configapi.GetClientGroupRequestObject{Name: "nope"})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.GetClientGroup404JSONResponse{}))
		})

		It("should delete", func() {
			_, err := h.PutClientGroup(ctx, configapi.PutClientGroupRequestObject{
				Name: "temp",
				Body: &configapi.ClientGroupInput{Clients: &[]string{}, Groups: &[]string{}},
			})
			Expect(err).Should(Succeed())

			resp, err := h.DeleteClientGroup(ctx, configapi.DeleteClientGroupRequestObject{Name: "temp"})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.DeleteClientGroup204Response{}))
		})

		It("should return 404 deleting nonexistent group", func() {
			resp, err := h.DeleteClientGroup(ctx, configapi.DeleteClientGroupRequestObject{Name: "ghost"})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.DeleteClientGroup404JSONResponse{}))
		})

		It("should reject empty client entry", func() {
			resp, err := h.PutClientGroup(ctx, configapi.PutClientGroupRequestObject{
				Name: "bad",
				Body: &configapi.ClientGroupInput{Clients: &[]string{"  "}, Groups: &[]string{}},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutClientGroup400JSONResponse{}))
		})
	})

	// --- Blocklist Sources ---

	Describe("BlocklistSources", func() {
		It("should create and list", func() {
			resp, err := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{
				Body: &configapi.BlocklistSourceInput{
					GroupName:  "ads",
					ListType:   configapi.BlocklistSourceInputListTypeDeny,
					SourceType: configapi.BlocklistSourceInputSourceTypeHttp,
					Source:     "https://example.com/list.txt",
					Enabled:    true,
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateBlocklistSource201JSONResponse{}))
			created := resp.(configapi.CreateBlocklistSource201JSONResponse)
			Expect(created.Id).ShouldNot(BeZero())

			listResp, err := h.ListBlocklistSources(ctx, configapi.ListBlocklistSourcesRequestObject{})
			Expect(err).Should(Succeed())
			list := listResp.(configapi.ListBlocklistSources200JSONResponse)
			Expect(list).Should(HaveLen(1))
		})

		It("should filter by group and type", func() {
			for _, s := range []configapi.BlocklistSourceInput{
				{GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny, SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://a.com", Enabled: true},
				{GroupName: "malware", ListType: configapi.BlocklistSourceInputListTypeDeny, SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://b.com", Enabled: true},
				{GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeAllow, SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://c.com", Enabled: true},
			} {
				body := s
				_, err := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{Body: &body})
				Expect(err).Should(Succeed())
			}

			groupName := "ads"
			resp, err := h.ListBlocklistSources(ctx, configapi.ListBlocklistSourcesRequestObject{
				Params: configapi.ListBlocklistSourcesParams{GroupName: &groupName},
			})
			Expect(err).Should(Succeed())
			Expect(resp.(configapi.ListBlocklistSources200JSONResponse)).Should(HaveLen(2))
		})

		It("should update", func() {
			body := configapi.BlocklistSourceInput{
				GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://old.com", Enabled: true,
			}
			createResp, _ := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{Body: &body})
			id := createResp.(configapi.CreateBlocklistSource201JSONResponse).Id

			newBody := configapi.BlocklistSourceInput{
				GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://new.com", Enabled: true,
			}
			resp, err := h.UpdateBlocklistSource(ctx, configapi.UpdateBlocklistSourceRequestObject{Id: id, Body: &newBody})
			Expect(err).Should(Succeed())
			Expect(resp.(configapi.UpdateBlocklistSource200JSONResponse).Source).Should(Equal("https://new.com"))
		})

		It("should return 404 updating nonexistent", func() {
			body := configapi.BlocklistSourceInput{
				GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://x.com", Enabled: true,
			}
			resp, err := h.UpdateBlocklistSource(ctx, configapi.UpdateBlocklistSourceRequestObject{Id: 9999, Body: &body})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.UpdateBlocklistSource404JSONResponse{}))
		})

		It("should delete", func() {
			body := configapi.BlocklistSourceInput{
				GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://x.com", Enabled: true,
			}
			createResp, _ := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{Body: &body})
			id := createResp.(configapi.CreateBlocklistSource201JSONResponse).Id

			resp, err := h.DeleteBlocklistSource(ctx, configapi.DeleteBlocklistSourceRequestObject{Id: id})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.DeleteBlocklistSource204Response{}))
		})

		It("should reject empty group name", func() {
			body := configapi.BlocklistSourceInput{
				GroupName: "", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeHttp, Source: "https://x.com", Enabled: true,
			}
			resp, err := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{Body: &body})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateBlocklistSource400JSONResponse{}))
		})

		It("should reject non-absolute file path", func() {
			body := configapi.BlocklistSourceInput{
				GroupName: "ads", ListType: configapi.BlocklistSourceInputListTypeDeny,
				SourceType: configapi.BlocklistSourceInputSourceTypeFile, Source: "relative/path.txt", Enabled: true,
			}
			resp, err := h.CreateBlocklistSource(ctx, configapi.CreateBlocklistSourceRequestObject{Body: &body})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateBlocklistSource400JSONResponse{}))
		})
	})

	// --- Custom DNS ---

	Describe("CustomDNSEntries", func() {
		It("should create and list", func() {
			resp, err := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "test.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "1.2.3.4", Ttl: 3600, Enabled: true,
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateCustomDNSEntry201JSONResponse{}))

			listResp, err := h.ListCustomDNSEntries(ctx, configapi.ListCustomDNSEntriesRequestObject{})
			Expect(err).Should(Succeed())
			Expect(listResp.(configapi.ListCustomDNSEntries200JSONResponse)).Should(HaveLen(1))
		})

		It("should reject invalid A record", func() {
			resp, err := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "test.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "not-an-ip", Ttl: 3600, Enabled: true,
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateCustomDNSEntry400JSONResponse{}))
		})

		It("should reject IPv6 for A record", func() {
			resp, err := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "test.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "::1", Ttl: 3600, Enabled: true,
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateCustomDNSEntry400JSONResponse{}))
		})

		It("should reject a TTL that would not fit in a uint32", func() {
			// The over-the-top case is only representable where int is 64 bits.
			rejected := []int{-1}
			if overflow := int64(math.MaxUint32) + 1; overflow <= math.MaxInt {
				rejected = append(rejected, int(overflow))
			}

			for _, ttl := range rejected {
				resp, err := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
					Body: &configapi.CustomDNSEntryInput{
						Domain: "test.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
						Value: "127.0.0.1", Ttl: ttl, Enabled: true,
					},
				})
				Expect(err).Should(Succeed())
				Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateCustomDNSEntry400JSONResponse{}))
			}
		})

		It("should reject IPv4 for AAAA record", func() {
			resp, err := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "test.local", RecordType: configapi.CustomDNSEntryInputRecordTypeAAAA,
					Value: "1.2.3.4", Ttl: 3600, Enabled: true,
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.CreateCustomDNSEntry400JSONResponse{}))
		})

		It("should update", func() {
			createResp, _ := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "old.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "1.2.3.4", Ttl: 3600, Enabled: true,
				},
			})
			id := createResp.(configapi.CreateCustomDNSEntry201JSONResponse).Id

			resp, err := h.UpdateCustomDNSEntry(ctx, configapi.UpdateCustomDNSEntryRequestObject{
				Id: id,
				Body: &configapi.CustomDNSEntryInput{
					Domain: "new.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "5.6.7.8", Ttl: 7200, Enabled: true,
				},
			})
			Expect(err).Should(Succeed())
			updated := resp.(configapi.UpdateCustomDNSEntry200JSONResponse)
			Expect(updated.Domain).Should(Equal("new.local"))
			Expect(updated.Value).Should(Equal("5.6.7.8"))
		})

		It("should delete", func() {
			createResp, _ := h.CreateCustomDNSEntry(ctx, configapi.CreateCustomDNSEntryRequestObject{
				Body: &configapi.CustomDNSEntryInput{
					Domain: "del.local", RecordType: configapi.CustomDNSEntryInputRecordTypeA,
					Value: "1.1.1.1", Ttl: 300, Enabled: true,
				},
			})
			id := createResp.(configapi.CreateCustomDNSEntry201JSONResponse).Id

			resp, err := h.DeleteCustomDNSEntry(ctx, configapi.DeleteCustomDNSEntryRequestObject{Id: id})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.DeleteCustomDNSEntry204Response{}))
		})
	})

	// --- Rebinding Settings ---

	Describe("RebindingSettings", func() {
		It("should get defaults", func() {
			resp, err := h.GetRebindingSettings(ctx, configapi.GetRebindingSettingsRequestObject{})
			Expect(err).Should(Succeed())
			rs := resp.(configapi.GetRebindingSettings200JSONResponse)
			Expect(rs.Enabled).Should(BeFalse())
			Expect(rs.AllowedDomains).Should(BeEmpty())
		})

		It("should persist an update", func() {
			_, err := h.PutRebindingSettings(ctx, configapi.PutRebindingSettingsRequestObject{
				Body: &configapi.RebindingSettingsInput{
					Enabled:        true,
					AllowedDomains: []string{"nas.example.com"},
				},
			})
			Expect(err).Should(Succeed())

			// PUT echoes the body, so read it back: what matters is what stuck
			resp, err := h.GetRebindingSettings(ctx, configapi.GetRebindingSettingsRequestObject{})
			Expect(err).Should(Succeed())
			rs := resp.(configapi.GetRebindingSettings200JSONResponse)
			Expect(rs.Enabled).Should(BeTrue())
			Expect(rs.AllowedDomains).Should(Equal([]string{"nas.example.com"}))
		})

		It("should serialize an empty allowlist as [] rather than null", func() {
			resp, err := h.GetRebindingSettings(ctx, configapi.GetRebindingSettingsRequestObject{})
			Expect(err).Should(Succeed())
			rs := resp.(configapi.GetRebindingSettings200JSONResponse)

			body, err := json.Marshal(rs)
			Expect(err).Should(Succeed())
			Expect(string(body)).Should(ContainSubstring(`"allowed_domains":[]`))
		})

		It("should reject a wildcard entry", func() {
			resp, err := h.PutRebindingSettings(ctx, configapi.PutRebindingSettingsRequestObject{
				Body: &configapi.RebindingSettingsInput{
					Enabled:        true,
					AllowedDomains: []string{"*.example.com"},
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutRebindingSettings400JSONResponse{}))
		})

		It("should reject an entry that differs only by case or trailing dot", func() {
			resp, err := h.PutRebindingSettings(ctx, configapi.PutRebindingSettingsRequestObject{
				Body: &configapi.RebindingSettingsInput{
					Enabled:        true,
					AllowedDomains: []string{"nas.example.com", "NAS.Example.com."},
				},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutRebindingSettings400JSONResponse{}))
		})
	})

	// --- HTTP/3 (DoH3) Settings ---

	Describe("HTTP3Settings", func() {
		// The handler reports the listener state of the running process, which it
		// cannot change, so each case needs its own handler.
		newHandler := func(rt configapi.DoH3Runtime) *configapi.ConfigHandler {
			return configapi.NewConfigHandler(store, reconf, rt, configapi.QueryLogHistorySource{})
		}

		get := func(h *configapi.ConfigHandler) configapi.HTTP3Settings {
			resp, err := h.GetHttp3Settings(ctx, configapi.GetHttp3SettingsRequestObject{})
			Expect(err).Should(Succeed())

			return configapi.HTTP3Settings(resp.(configapi.GetHttp3Settings200JSONResponse))
		}

		put := func(h *configapi.ConfigHandler, enabled bool) configapi.PutHttp3SettingsResponseObject {
			resp, err := h.PutHttp3Settings(ctx, configapi.PutHttp3SettingsRequestObject{
				Body: &configapi.HTTP3SettingsInput{Enabled: enabled},
			})
			Expect(err).Should(Succeed())

			return resp
		}

		It("should get defaults on a database with no row", func() {
			hs := get(newHandler(configapi.DoH3Runtime{}))
			Expect(hs.Enabled).Should(BeFalse())
			Expect(hs.Active).Should(BeFalse())
			Expect(hs.RestartRequired).Should(BeFalse())
			Expect(hs.UnavailableReason).Should(BeNil())
		})

		It("should persist an update", func() {
			h3 := newHandler(configapi.DoH3Runtime{})
			put(h3, true)

			// PUT echoes the body, so read it back: what matters is what stuck
			Expect(get(h3).Enabled).Should(BeTrue())
		})

		// gorm's Save skips zero-valued fields, which would make the toggle
		// one-way: turning DoH3 off would report success and persist nothing.
		It("should persist turning the toggle back off", func() {
			h3 := newHandler(configapi.DoH3Runtime{})
			put(h3, true)
			put(h3, false)

			Expect(get(h3).Enabled).Should(BeFalse())
		})

		It("should ask for a restart when the stored setting and the listener disagree", func() {
			h3 := newHandler(configapi.DoH3Runtime{})

			resp := put(h3, true)
			hs := configapi.HTTP3Settings(resp.(configapi.PutHttp3Settings200JSONResponse))
			Expect(hs.Enabled).Should(BeTrue())
			Expect(hs.Active).Should(BeFalse())
			Expect(hs.RestartRequired).Should(BeTrue())
		})

		It("should not ask for a restart when the listener already matches", func() {
			h3 := newHandler(configapi.DoH3Runtime{Active: true})
			put(h3, true)

			hs := get(h3)
			Expect(hs.Active).Should(BeTrue())
			Expect(hs.RestartRequired).Should(BeFalse())
		})

		// A restart cannot bind a listener that has nowhere to bind, so promising
		// one would send the operator to bounce the process for nothing.
		It("should report the blocking reason instead of asking for a restart", func() {
			h3 := newHandler(configapi.DoH3Runtime{UnavailableReason: "ports.https is empty"})
			put(h3, true)

			hs := get(h3)
			Expect(hs.Enabled).Should(BeTrue())
			Expect(hs.Active).Should(BeFalse())
			Expect(hs.RestartRequired).Should(BeFalse())
			Expect(hs.UnavailableReason).ShouldNot(BeNil())
			Expect(*hs.UnavailableReason).Should(Equal("ports.https is empty"))
		})

		It("should reject a missing body", func() {
			resp, err := newHandler(configapi.DoH3Runtime{}).
				PutHttp3Settings(ctx, configapi.PutHttp3SettingsRequestObject{})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutHttp3Settings400JSONResponse{}))
		})

		// Reconfigure rebuilds the resolver chain and cannot rebind a listener, so
		// running it here would report success while changing nothing.
		It("should not reconfigure", func() {
			put(newHandler(configapi.DoH3Runtime{}), true)
			Expect(reconf.calls).Should(BeZero())
		})
	})

	// --- Block Settings ---

	Describe("BlockSettings", func() {
		It("should get defaults", func() {
			resp, err := h.GetBlockSettings(ctx, configapi.GetBlockSettingsRequestObject{})
			Expect(err).Should(Succeed())
			bs := resp.(configapi.GetBlockSettings200JSONResponse)
			Expect(bs.BlockType).Should(Equal("ZEROIP"))
			Expect(bs.BlockTtl).Should(Equal("6h"))
		})

		It("should update", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "NXDOMAIN", BlockTtl: "30m"},
			})
			Expect(err).Should(Succeed())
			bs := resp.(configapi.PutBlockSettings200JSONResponse)
			Expect(bs.BlockType).Should(Equal("NXDOMAIN"))
		})

		It("should store the refused block type, not just echo it", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "REFUSED", BlockTtl: "1h"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings200JSONResponse{}))

			// PUT echoes the request body back, so read it again: what matters is
			// that REFUSED reached the store, which is the only thing
			// BuildBlockingConfig (and therefore the resolver) ever looks at.
			got, err := h.GetBlockSettings(ctx, configapi.GetBlockSettingsRequestObject{})
			Expect(err).Should(Succeed())
			Expect(got.(configapi.GetBlockSettings200JSONResponse).BlockType).Should(Equal("REFUSED"))
		})

		It("should accept a comma-separated list of block IPs", func() {
			// resolver.createBlockHandler builds an ipBlockHandler from this form,
			// and the store is the only writer of blockType, so rejecting it here
			// would make the documented v4+v6 pair unreachable.
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "192.0.2.10, 2001:db8::1", BlockTtl: "1h"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings200JSONResponse{}))
		})

		It("should reject a list with one unparseable member", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "192.0.2.10, nope", BlockTtl: "1h"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings400JSONResponse{}))
		})

		It("should reject invalid block type", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "INVALID", BlockTtl: "1h"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings400JSONResponse{}))
		})

		It("should reject invalid TTL", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "ZEROIP", BlockTtl: "bad"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings400JSONResponse{}))
		})

		It("should accept IP address as block type", func() {
			resp, err := h.PutBlockSettings(ctx, configapi.PutBlockSettingsRequestObject{
				Body: &configapi.BlockSettingsInput{BlockType: "192.168.1.1", BlockTtl: "1h"},
			})
			Expect(err).Should(Succeed())
			Expect(resp).Should(BeAssignableToTypeOf(configapi.PutBlockSettings200JSONResponse{}))
		})
	})

	// --- Apply ---

	Describe("Apply", func() {
		It("should return ok on success", func() {
			resp, err := h.ApplyConfig(ctx, configapi.ApplyConfigRequestObject{})
			Expect(err).Should(Succeed())
			result := resp.(configapi.ApplyConfig200JSONResponse)
			Expect(result.Status).Should(Equal(configapi.Ok))
			Expect(result.Message).Should(ContainSubstring("applied successfully"))
		})

		It("should return 500 on reconfigure failure", func() {
			reconf.err = errors.New("chain build failed")

			resp, err := h.ApplyConfig(ctx, configapi.ApplyConfigRequestObject{})
			Expect(err).Should(Succeed())
			result := resp.(configapi.ApplyConfig500JSONResponse)
			Expect(result.Status).Should(Equal(configapi.Error))
			Expect(result.Error).ShouldNot(BeNil())
			Expect(*result.Error).Should(ContainSubstring("chain build failed"))
		})
	})
})
