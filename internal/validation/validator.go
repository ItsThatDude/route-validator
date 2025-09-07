package validation

import (
	"fmt"
	"sort"
	"strings"

	"antware.xyz/route-validator/internal/config"
	v1 "k8s.io/api/core/v1"
)

type Validator struct {
	Config *config.ValidatorConfig
}

func (v *Validator) GetRequiredSubdomain(namespace *v1.Namespace) string {
	subdomain := namespace.Name

	if v.Config.SubdomainLabel != "" {
		if val, ok := namespace.Labels[v.Config.SubdomainLabel]; ok {
			subdomain = val
		}
	}

	return subdomain
}

func MatchesAnyDomain(hostnames []string, domains []string) bool {
	for _, host := range hostnames {
		matchedDomain := FindFirstDomain(host, domains)

		if matchedDomain != "" {
			return true
		}
	}

	return false
}

func FindFirstDomain(hostname string, domains []string) string {
	// Sort domains by length (longest first) so more specific ones are checked first
	sort.Slice(domains, func(i, j int) bool {
		return len(domains[i]) > len(domains[j])
	})

	for _, domain := range domains {
		if strings.HasSuffix(hostname, domain) {
			if strings.HasPrefix(domain, ".") {
				return domain
			}
			return "." + domain
		}
	}
	return ""
}

func (v *Validator) ValidateHostnames(namespace *v1.Namespace, hostnames []string) (bool, error) {
	subdomain := v.GetRequiredSubdomain(namespace)

	for _, host := range hostnames {
		matchedDomain := FindFirstDomain(host, v.Config.MatchDomains)
		if matchedDomain != "" {
			domain := subdomain + matchedDomain
			result := strings.HasSuffix(host, "."+domain)

			if !result {
				return false, fmt.Errorf("hostname %s must have subdomain %s for domain %s", host, subdomain, matchedDomain)
			}
		}
	}

	return true, nil
}
