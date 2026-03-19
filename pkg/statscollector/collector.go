package statscollector

import (
	"sort"
	"sync"
	"time"
)

const (
	BucketCount    = 144 // 24h / 10min
	BucketDuration = 10 * time.Minute
	DefaultTopN    = 10
)

// QueryRecord is the data captured per DNS query.
type QueryRecord struct {
	Timestamp    time.Time
	Client       string // client name or IP
	Domain       string // question domain (lowercase, no trailing dot)
	QueryType    string // A, AAAA, SRV, MX, etc.
	ResponseType string // RESOLVED, CACHED, BLOCKED, CUSTOMDNS, etc.
}

// TimeBucket holds aggregated counts for one 10-minute window.
type TimeBucket struct {
	Timestamp    time.Time      `json:"ts"`
	Total        int            `json:"total"`
	Blocked      int            `json:"blocked"`
	ClientCounts map[string]int `json:"clients,omitempty"`
}

// DomainCount is a domain name with its hit count.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// ClientCount is a client identifier with its query count.
type ClientCount struct {
	Client string `json:"client"`
	Count  int    `json:"count"`
}

// Collector accumulates query stats in memory using a ring buffer.
type Collector struct {
	mu      sync.RWMutex
	buckets [BucketCount]TimeBucket
	head    int // index of current bucket

	// Running counters (since startup)
	domainPermitted map[string]int
	domainBlocked   map[string]int
	clientTotal     map[string]int
	clientBlocked   map[string]int
	queryTypes      map[string]int
	responseTypes   map[string]int

	// Allow injecting time for testing
	now func() time.Time
}

// Option configures a Collector.
type Option func(*Collector)

// WithClock overrides the time source (useful for testing).
func WithClock(fn func() time.Time) Option {
	return func(c *Collector) { c.now = fn }
}

// New creates a Collector ready to receive query records.
func New(opts ...Option) *Collector {
	c := &Collector{
		domainPermitted: make(map[string]int),
		domainBlocked:   make(map[string]int),
		clientTotal:     make(map[string]int),
		clientBlocked:   make(map[string]int),
		queryTypes:      make(map[string]int),
		responseTypes:   make(map[string]int),
		now:             time.Now,
	}

	for _, o := range opts {
		o(c)
	}

	now := c.now().Truncate(BucketDuration)
	c.buckets[0] = TimeBucket{
		Timestamp:    now,
		ClientCounts: make(map[string]int),
	}

	return c
}

// Record ingests a single query observation.
func (c *Collector) Record(q QueryRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.advanceBuckets()

	b := &c.buckets[c.head]
	b.Total++

	if q.ResponseType == "BLOCKED" {
		b.Blocked++
	}

	if b.ClientCounts == nil {
		b.ClientCounts = make(map[string]int)
	}

	b.ClientCounts[q.Client]++

	if q.ResponseType == "BLOCKED" {
		c.domainBlocked[q.Domain]++
		c.clientBlocked[q.Client]++
	} else {
		c.domainPermitted[q.Domain]++
	}

	c.clientTotal[q.Client]++
	c.queryTypes[q.QueryType]++
	c.responseTypes[q.ResponseType]++
}

// advanceBuckets moves the ring head forward to cover the current time window.
// Caller must hold c.mu.
func (c *Collector) advanceBuckets() {
	now := c.now().Truncate(BucketDuration)

	for c.buckets[c.head].Timestamp.Before(now) {
		prev := c.buckets[c.head].Timestamp
		c.head = (c.head + 1) % BucketCount

		c.buckets[c.head] = TimeBucket{
			Timestamp:    prev.Add(BucketDuration),
			ClientCounts: make(map[string]int),
		}

		if !c.buckets[c.head].Timestamp.Before(now) {
			break
		}
	}
}

// OverTime returns all 144 buckets in chronological order (oldest first).
func (c *Collector) OverTime() []TimeBucket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]TimeBucket, BucketCount)
	for i := range BucketCount {
		idx := (c.head + 1 + i) % BucketCount
		out[i] = c.buckets[idx]

		// Deep-copy the client map
		if c.buckets[idx].ClientCounts != nil {
			m := make(map[string]int, len(c.buckets[idx].ClientCounts))
			for k, v := range c.buckets[idx].ClientCounts {
				m[k] = v
			}

			out[i].ClientCounts = m
		}
	}

	return out
}

// QueryTypes returns a copy of the query type distribution.
func (c *Collector) QueryTypes() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return copyMap(c.queryTypes)
}

// ResponseTypes returns a copy of the response type distribution.
func (c *Collector) ResponseTypes() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return copyMap(c.responseTypes)
}

// TopDomains returns the top-N permitted and blocked domains by hit count.
func (c *Collector) TopDomains(n int) (permitted, blocked []DomainCount) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	permitted = topN(c.domainPermitted, n, func(k string, v int) DomainCount {
		return DomainCount{Domain: k, Count: v}
	})

	blocked = topN(c.domainBlocked, n, func(k string, v int) DomainCount {
		return DomainCount{Domain: k, Count: v}
	})

	return permitted, blocked
}

// TopClients returns the top-N clients by total and blocked query count.
func (c *Collector) TopClients(n int) (total, blocked []ClientCount) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total = topN(c.clientTotal, n, func(k string, v int) ClientCount {
		return ClientCount{Client: k, Count: v}
	})

	blocked = topN(c.clientBlocked, n, func(k string, v int) ClientCount {
		return ClientCount{Client: k, Count: v}
	})

	return total, blocked
}

// ActiveClients returns the number of unique clients seen.
func (c *Collector) ActiveClients() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.clientTotal)
}

// TotalQueries returns the cumulative total and blocked query counts.
func (c *Collector) TotalQueries() (total, blocked int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, v := range c.clientTotal {
		total += v
	}

	for _, v := range c.clientBlocked {
		blocked += v
	}

	return total, blocked
}

func copyMap(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}

// topN extracts the top-n entries from a map, sorted descending by value.
func topN[T any](m map[string]int, n int, conv func(string, int) T) []T {
	type kv struct {
		key string
		val int
	}

	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{k, v})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].val > items[j].val
	})

	if n > len(items) {
		n = len(items)
	}

	out := make([]T, n)
	for i := range n {
		out[i] = conv(items[i].key, items[i].val)
	}

	return out
}
