package main

import (
	"fmt"
	"os"
	"strings"
)

// uses Traefik-compliant notation to set up HTTP ingress to route traffic to the dev container
func setupDevIngress(
	builder *BuilderSpec,
	ingressSettings devIngressSettings,
	bobfile *Bobfile,
) ([]string, string) {
	if builder.DevHttpIngress == "" {
		return nil, ""
	}

	containerPort := builder.DevHttpIngress

	if err := ingressSettings.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: cannot setup DevHttpIngress because user's config: %v\n", err)
		return nil, ""
	}

	// "joonas.fi-blog" => "joonas-fi-blog"
	// don't accept multi-level subdomains because wildcard TLS certs only apply
	// one level.
	ingressAppId := strings.ReplaceAll(bobfile.ProjectName, ".", "-")
	if builder.Name != "default" {
		ingressAppId += "-" + builder.Name
	}

	ingressHostname := ingressAppId + "." + ingressSettings.Domain

	labels := []string{
		// Edgerouter needs explicit opt-in for "no auth" to not accidentally expose private services
		"edgerouter.auth=public",
		"traefik.frontend.rule=Host:" + ingressHostname,
		"traefik.port=" + containerPort,
	}

	if containerPort == "443" {
		labels = append(
			labels,
			"traefik.protocol=https",
			"traefik.backend.tls.insecureSkipVerify=true")
	}

	dockerCmd := []string{}

	for _, label := range labels {
		dockerCmd = append(dockerCmd, "--label", label)
	}

	if ingressSettings.DockerNetwork != "" {
		dockerCmd = append(dockerCmd, "--network", ingressSettings.DockerNetwork)
	}

	return dockerCmd, ingressHostname
}
