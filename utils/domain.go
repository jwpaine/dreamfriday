package utils

import "strings"

const BaseDomain = "dreamfriday.com"

// GetSubdomain extracts all subdomains before the base domain.
func GetSubdomain(domain string) string {
	if domain == "localhost:8081" || domain == BaseDomain {
		return BaseDomain
	}

	// Ensure the domain ends with baseDomain
	if !strings.HasSuffix(domain, "."+BaseDomain) {
		return BaseDomain
	}

	// Remove the base domain from the full domain
	subdomain := strings.TrimSuffix(domain, "."+BaseDomain)

	// Ensure there are no empty subdomains (e.g., "sub..example.com")
	if strings.Contains(subdomain, "..") || subdomain == "" {
		return BaseDomain
	}

	return subdomain
}

func SiteDomain(site string) string {
	if site == "" {
		return BaseDomain
	}
	if site == BaseDomain {
		return BaseDomain
	}
	return site + "." + BaseDomain
}
