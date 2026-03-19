package advertise

import (
	"fmt"
	"net"

	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/log"
	"github.com/miekg/dns"
)

// ResolveAddress determines the IP to advertise for client group endpoint domains.
// Returns nil if advertiseAddress is empty (feature disabled).
func ResolveAddress(addr string) (net.IP, error) {
	if addr == "" {
		return nil, nil
	}

	if addr != "auto" {
		ip := net.ParseIP(addr)
		if ip == nil {
			return nil, fmt.Errorf("invalid advertiseAddress %q", addr)
		}

		return ip, nil
	}

	// Auto-detect: try k8s first, then outbound interface
	if isKubernetes() {
		ip, err := detectKubernetesIP()
		if err != nil {
			log.Log().Warnf("k8s LB IP detection failed, falling back to outbound interface: %v", err)
		} else {
			return ip, nil
		}
	}

	return detectOutboundIP()
}

// InjectRecords merges auto-advertised A/AAAA records into a CustomDNS mapping.
// User-defined entries for the same domain are not overwritten.
func InjectRecords(mapping config.CustomDNSMapping, domains []string, ip net.IP, ttl uint32) {
	for _, domain := range domains {
		injectForDomain(mapping, domain, ip, ttl)
		injectForDomain(mapping, "*."+domain, ip, ttl)
	}
}

func injectForDomain(mapping config.CustomDNSMapping, domain string, ip net.IP, ttl uint32) {
	fqdn := dns.Fqdn(domain)

	if _, exists := mapping[fqdn]; exists {
		log.Log().Debugf("skipping auto-advertise for %s (user-defined entry exists)", fqdn)

		return
	}

	rr := makeRR(fqdn, ip, ttl)
	mapping[fqdn] = config.CustomDNSEntries{rr}

	log.Log().Infof("auto-advertise: %s → %s", fqdn, ip)
}

func makeRR(fqdn string, ip net.IP, ttl uint32) dns.RR {
	if ip.To4() != nil {
		return &dns.A{
			Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
			A:   ip.To4(),
		}
	}

	return &dns.AAAA{
		Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
		AAAA: ip,
	}
}
