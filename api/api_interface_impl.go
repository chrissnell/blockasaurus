// Modified by Chris Snell, 2026
// SPDX-License-Identifier: Apache-2.0

//go:generate go tool oapi-codegen --config=types.cfg.yaml ../docs/api/openapi.yaml
//go:generate go tool oapi-codegen --config=server.cfg.yaml ../docs/api/openapi.yaml
//go:generate go tool oapi-codegen --config=client.cfg.yaml ../docs/api/openapi.yaml

package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/0xERR0R/blocky/log"
	"github.com/0xERR0R/blocky/model"
	"github.com/0xERR0R/blocky/stats"
	"github.com/0xERR0R/blocky/util"
	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"
)

type httpReqCtxKey struct{}

// BlockingStatus represents the current blocking status
type BlockingStatus struct {
	// True if blocking is enabled
	Enabled bool
	// Disabled group names
	DisabledGroups []string
	// If blocking is temporarily disabled: amount of seconds until blocking will be enabled
	AutoEnableInSec int
}

// BlockingControl interface to control the blocking status
type BlockingControl interface {
	EnableBlocking(ctx context.Context)
	DisableBlocking(ctx context.Context, duration time.Duration, disableGroups []string) error
	BlockingStatus() BlockingStatus
}

// ListRefresher interface to control the list refresh
type ListRefresher interface {
	RefreshLists(ctx context.Context) error
}

type Querier interface {
	Query(
		ctx context.Context, serverHost string, clientIP net.IP, question string, qType dns.Type,
	) (*model.Response, error)
}

type CacheControl interface {
	FlushCaches(ctx context.Context)
}

<<<<<<< HEAD
// ResolverAccessor provides dynamic access to resolver capabilities.
// The server implements this to resolve through the current (possibly hot-swapped) chain.
type ResolverAccessor interface {
	BlockingControl() (BlockingControl, error)
	ListRefresher() (ListRefresher, error)
	CacheControl() (CacheControl, error)
=======
// StatsProvider exposes the in-memory statistics snapshot.
type StatsProvider interface {
	// StatsEnabled reports whether statistics collection is active.
	StatsEnabled() bool
	// Stats returns the current snapshot.
	Stats() stats.Result
>>>>>>> upstream/main
}

func RegisterOpenAPIEndpoints(router chi.Router, impl StrictServerInterface) {
	middleware := []StrictMiddlewareFunc{ctxWithHTTPRequestMiddleware}

	HandlerFromMuxWithBaseURL(NewStrictHandler(impl, middleware), router, "/api")
}

func ctxWithHTTPRequestMiddleware(handler StrictHandlerFunc, operationID string) StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
		ctx = context.WithValue(ctx, httpReqCtxKey{}, r)

		return handler(ctx, w, r, request)
	}
}

type OpenAPIInterfaceImpl struct {
<<<<<<< HEAD
	resolver ResolverAccessor
	querier  Querier
}

func NewOpenAPIInterfaceImpl(resolver ResolverAccessor, querier Querier) *OpenAPIInterfaceImpl {
	return &OpenAPIInterfaceImpl{
		resolver: resolver,
		querier:  querier,
=======
	control      BlockingControl
	querier      Querier
	refresher    ListRefresher
	cacheControl CacheControl
	stats        StatsProvider
}

func NewOpenAPIInterfaceImpl(control BlockingControl,
	querier Querier,
	refresher ListRefresher,
	cacheControl CacheControl,
	statsProvider StatsProvider,
) *OpenAPIInterfaceImpl {
	return &OpenAPIInterfaceImpl{
		control:      control,
		querier:      querier,
		refresher:    refresher,
		cacheControl: cacheControl,
		stats:        statsProvider,
>>>>>>> upstream/main
	}
}

func (i *OpenAPIInterfaceImpl) DisableBlocking(ctx context.Context,
	request DisableBlockingRequestObject,
) (DisableBlockingResponseObject, error) {
	control, err := i.resolver.BlockingControl()
	if err != nil {
		return DisableBlocking400TextResponse(log.EscapeInput(err.Error())), nil
	}

	var (
		duration time.Duration
		groups   []string
	)

	if request.Params.Duration != nil {
		duration, err = time.ParseDuration(*request.Params.Duration)
		if err != nil {
			return DisableBlocking400TextResponse(log.EscapeInput(err.Error())), nil
		}
	}

	if request.Params.Groups != nil && len(*request.Params.Groups) > 0 {
		groups = strings.Split(*request.Params.Groups, ",")
	}

	err = control.DisableBlocking(ctx, duration, groups)
	if err != nil {
		return DisableBlocking400TextResponse(log.EscapeInput(err.Error())), nil
	}

	return DisableBlocking200Response{}, nil
}

func (i *OpenAPIInterfaceImpl) EnableBlocking(ctx context.Context, _ EnableBlockingRequestObject,
) (EnableBlockingResponseObject, error) {
	control, err := i.resolver.BlockingControl()
	if err != nil {
		return EnableBlocking200Response{}, fmt.Errorf("blocking control: %w", err)
	}

	control.EnableBlocking(ctx)

	return EnableBlocking200Response{}, nil
}

func (i *OpenAPIInterfaceImpl) BlockingStatus(_ context.Context, _ BlockingStatusRequestObject,
) (BlockingStatusResponseObject, error) {
	control, err := i.resolver.BlockingControl()
	if err != nil {
		return nil, fmt.Errorf("blocking control: %w", err)
	}

	blStatus := control.BlockingStatus()

	result := ApiBlockingStatus{
		Enabled: blStatus.Enabled,
	}

	if blStatus.AutoEnableInSec > 0 {
		result.AutoEnableInSec = &blStatus.AutoEnableInSec
	}

	if len(blStatus.DisabledGroups) > 0 {
		result.DisabledGroups = &blStatus.DisabledGroups
	}

	return BlockingStatus200JSONResponse(result), nil
}

func (i *OpenAPIInterfaceImpl) ListRefresh(ctx context.Context,
	_ ListRefreshRequestObject,
) (ListRefreshResponseObject, error) {
	refresher, err := i.resolver.ListRefresher()
	if err != nil {
		return ListRefresh500TextResponse(log.EscapeInput(err.Error())), nil
	}

	err = refresher.RefreshLists(ctx)
	if err != nil {
		return ListRefresh500TextResponse(log.EscapeInput(err.Error())), nil
	}

	return ListRefresh200Response{}, nil
}

func (i *OpenAPIInterfaceImpl) Query(ctx context.Context, request QueryRequestObject) (QueryResponseObject, error) {
	qType := dns.Type(dns.StringToType[request.Body.Type])
	if qType == dns.Type(dns.TypeNone) {
		return Query400TextResponse(fmt.Sprintf("unknown query type '%s'", request.Body.Type)), nil
	}

	var (
		serverHost string
		clientIP   net.IP
	)

	httpReq, ok := ctx.Value(httpReqCtxKey{}).(*http.Request)
	if ok {
		serverHost = httpReq.Host
		clientIP = util.HTTPClientIP(httpReq)
	}

	resp, err := i.querier.Query(ctx, serverHost, clientIP, dns.Fqdn(request.Body.Query), qType)
	if err != nil {
		return nil, fmt.Errorf("query failed for '%s' (type %s): %w", request.Body.Query, request.Body.Type, err)
	}

	return Query200JSONResponse(ApiQueryResult{
		Reason:       resp.Reason,
		ResponseType: resp.RType.String(),
		Response:     util.AnswerToString(resp.Res.Answer),
		ReturnCode:   dns.RcodeToString[resp.Res.Rcode],
	}), nil
}

func (i *OpenAPIInterfaceImpl) CacheFlush(ctx context.Context,
	_ CacheFlushRequestObject,
) (CacheFlushResponseObject, error) {
	cc, err := i.resolver.CacheControl()
	if err != nil {
		return nil, fmt.Errorf("cache control: %w", err)
	}

	cc.FlushCaches(ctx)

	return CacheFlush200Response{}, nil
}

func (i *OpenAPIInterfaceImpl) GetStats(_ context.Context, _ GetStatsRequestObject,
) (GetStatsResponseObject, error) {
	if i.stats == nil || !i.stats.StatsEnabled() {
		return GetStats503TextResponse("statistics are disabled"), nil
	}

	return GetStats200JSONResponse(toAPIStats(i.stats.Stats())), nil
}

func toAPIStats(r stats.Result) ApiStats {
	return ApiStats{
		Start: r.Start,
		End:   r.End,
		Summary: ApiStatsSummary{
			Queries:       r.Summary.Queries,
			Cached:        r.Summary.Cached,
			Forwarded:     r.Summary.Forwarded,
			Blocked:       r.Summary.Blocked,
			Filtered:      r.Summary.Filtered,
			Local:         r.Summary.Local,
			Dropped:       r.Summary.Dropped,
			Errors:        r.Summary.Errors,
			AvgResponseMs: r.Summary.AvgResponseMs,
			CacheHitRate:  r.Summary.CacheHitRate,
		},
		ByResponseType:    r.ByResponseType,
		ByQueryType:       r.ByQueryType,
		ByResponseCode:    r.ByResponseCode,
		PerHour:           toAPIHourPoints(r.PerHour),
		TopDomains:        toAPINameCounts(r.TopDomains),
		TopBlockedDomains: toAPINameCounts(r.TopBlockedDomains),
		TopClients:        toAPINameCounts(r.TopClients),
		Lists: ApiListCounts{
			Denylist:  r.Lists.Denylist,
			Allowlist: r.Lists.Allowlist,
		},
		Cache: ApiCacheStats{Entries: r.CacheEntries},
	}
}

func toAPINameCounts(in []stats.NameCount) []ApiNameCount {
	out := make([]ApiNameCount, 0, len(in))
	for _, nc := range in {
		out = append(out, ApiNameCount{Name: nc.Name, Count: nc.Count})
	}

	return out
}

func toAPIHourPoints(in []stats.HourPoint) []ApiHourPoint {
	out := make([]ApiHourPoint, 0, len(in))
	for _, p := range in {
		out = append(out, ApiHourPoint{
			Hour: p.Hour, Queries: p.Queries, Blocked: p.Blocked, Filtered: p.Filtered,
		})
	}

	return out
}
