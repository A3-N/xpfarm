package core

import (
	"net"
	"net/url"
	"strings"
	"xpfarm/pkg/utils"
)

type TargetType string

const (
	TargetTypeIP     TargetType = "ip"
	TargetTypeCIDR   TargetType = "cidr"
	TargetTypeDomain TargetType = "domain"
	TargetTypeURL    TargetType = "url"
)

type ParsedTarget struct {
	Value string
	Type  TargetType
}

// ParseTarget determines the type of the target string.
// It does NOT perform DNS resolution; that's for a separate step or the tools themselves.
func ParseTarget(input string) ParsedTarget {
	input = strings.TrimSpace(input)

	// Check if CIDR
	if _, _, err := net.ParseCIDR(input); err == nil {
		return ParsedTarget{Value: input, Type: TargetTypeCIDR}
	}

	// Check if IP
	if net.ParseIP(input) != nil {
		return ParsedTarget{Value: input, Type: TargetTypeIP}
	}

	// Check if URL (has scheme)
	if strings.Contains(input, "://") {
		u, err := url.Parse(input)
		if err == nil && u.Scheme != "" && u.Host != "" {
			return ParsedTarget{Value: input, Type: TargetTypeURL}
		}
	}

	// Default to Domain
	return ParsedTarget{Value: input, Type: TargetTypeDomain}
}

// TargetCheckResult holds intelligence data
type TargetCheckResult struct {
	IsCloudflare bool
	IsAlive      bool
	Status       string
	ResolvedIPs  []string
}

// ResolveAndCheck performs DNS resolution and checks Cloudflare status and Liveness
func ResolveAndCheck(input string) TargetCheckResult {
	parsed := ParseTarget(input)
	res := TargetCheckResult{
		Status:  "down",
		IsAlive: false,
	}

	ips := []string{}

	// 1. Resolve IP
	switch parsed.Type {
	case TargetTypeIP:
		ips = append(ips, parsed.Value)
		res.IsAlive = true // Assume IP provided is "alive" in terms of resolution
	case TargetTypeDomain, TargetTypeURL:
		host := parsed.Value
		if parsed.Type == TargetTypeURL {
			u, _ := url.Parse(parsed.Value)
			host = u.Host
		}
		// Strip port
		if strings.Contains(host, ":") {
			h, _, err := net.SplitHostPort(host)
			if err == nil {
				host = h
			}
		}

		resolved, err := net.LookupHost(host)
		if err == nil && len(resolved) > 0 {
			ips = append(ips, resolved...)
			res.IsAlive = true
			res.Status = "up"
		} else {
			res.Status = "unreachable"
			return res
		}
	case TargetTypeCIDR:
		ip, _, err := net.ParseCIDR(parsed.Value)
		if err == nil {
			ips = append(ips, ip.String())
			res.IsAlive = true
			res.Status = "up"
		}
	}

	// 2. Check Cloudflare
	for _, ip := range ips {
		if utils.IsCloudflareIP(ip) {
			res.IsCloudflare = true
			break
		}
	}

	res.ResolvedIPs = ips
	if res.IsAlive {
		res.Status = "up"
	}
	if len(ips) == 0 {
		res.IsAlive = false
		res.Status = "unreachable"
	}

	return res
}
