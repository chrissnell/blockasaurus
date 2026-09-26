// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/pkg/statscollector"
	"github.com/0xERR0R/blocky/resolver"

	"github.com/creasty/defaults"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The dashboard's statistics are fed by exactly one assignment in
// createQueryResolver — metricsResolver.StatsCollector. Nothing else in the
// tree notices if it goes: MetricsResolver works without it, every resolver
// spec passes without it, and the /api/stats/* handlers answer with empty
// series rather than an error. An upstream merge resolving that hunk toward
// upstream (which has no such field) would silently empty the dashboard, which
// is the class of loss docs/UPSTREAM_SYNC.md §3a exists to turn into a test
// result.
//
// Same reasoning for the resolvers upstream added in this sync: they merge at
// upstream defaults and are inert when disabled, so a chain that dropped them
// behaves identically until someone turns the feature on and finds the flag
// does nothing.
var _ = Describe("Resolver chain wiring", func() {
	var (
		cfg       config.Config
		collector *statscollector.Collector
		chain     resolver.ChainedResolver
	)

	BeforeEach(func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		Expect(defaults.Set(&cfg)).Should(Succeed())

		cfg.Upstreams.Groups = map[string][]config.Upstream{
			"default": {config.Upstream{Net: config.NetProtocolTcpUdp, Host: "1.1.1.1", Port: 53}},
		}

		collector = statscollector.New()
		DeferCleanup(collector.Close)

		bootstrap, err := resolver.NewBootstrap(ctx, &cfg)
		Expect(err).Should(Succeed())

		chain, err = createQueryResolver(ctx, &cfg, bootstrap, nil, nil, collector)
		Expect(err).Should(Succeed())
	})

	It("hands the stats collector to the metrics resolver", func() {
		metricsResolver, err := resolver.GetFromChainWithType[*resolver.MetricsResolver](chain)
		Expect(err).Should(Succeed())
		Expect(metricsResolver.StatsCollector).Should(BeIdenticalTo(collector))
	})

	DescribeTable("chains the resolver",
		func(present func(resolver.ChainedResolver) bool) {
			Expect(present(chain)).Should(BeTrue())
		},
		Entry("ECS client", inChain[*resolver.ECSClientResolver]),
		Entry("rate limiting", inChain[*resolver.RateLimitingResolver]),
		Entry("rebinding protection", inChain[*resolver.RebindingProtectionResolver]),
		Entry("query logging", inChain[*resolver.QueryLoggingResolver]),
		Entry("blocking", inChain[*resolver.BlockingResolver]),
		Entry("caching", inChain[*resolver.CachingResolver]),
	)

	// Upstream's own statistics resolver was dropped in favour of the fork's
	// collector (docs/UPSTREAM_SYNC.md D1). Both register GET /api/stats, and
	// chi panics on a duplicate pattern, so a merge that re-chained it would
	// not be a subtle regression.
	It("does not chain a second statistics subsystem", func() {
		var found bool

		resolver.ForEach(chain, func(r resolver.Resolver) {
			if resolver.Name(r) == "stats" {
				found = true
			}
		})

		Expect(found).Should(BeFalse())
	})
})

func inChain[T resolver.ChainedResolver](chain resolver.ChainedResolver) bool {
	_, err := resolver.GetFromChainWithType[T](chain)

	return err == nil
}
