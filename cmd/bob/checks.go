package main

import (
	"fmt"
)

func RunChecks(bobfile *Bobfile) ([]CheckResult, error) {
	ctx := &CheckContext{
		Bobfile: bobfile,
		Results: []CheckResult{},
	}

	checks := []func(*CheckContext) error{
		dockerRegistryCredentialsPresent,
		passableEnvVarsPresent,
		licensePresent,
		readmePresent,
	}

	for _, check := range checks {
		if err := check(ctx); err != nil {
			return nil, err
		}
	}

	return ctx.Results, nil
}

func licensePresent(ctx *CheckContext) error {
	licenseCheck := ctx.NewCheck("License present")

	exists, errChecking := fileExists("LICENSE")
	if errChecking != nil {
		return errChecking
	}

	if exists {
		return licenseCheck.Ok()
	} else {
		return licenseCheck.Fail("Project must have a LICENSE file")
	}
}

func readmePresent(ctx *CheckContext) error {
	readmeCheck := ctx.NewCheck("Readme present")

	exists, errChecking := fileExists("README.md")
	if errChecking != nil {
		return errChecking
	}

	if exists {
		return readmeCheck.Ok()
	} else {
		return readmeCheck.Fail("Project must have a README.md file")
	}
}

func dockerRegistryCredentialsPresent(ctx *CheckContext) error {
	registryCredentials := ctx.NewCheck("Docker registry credentials")

	if len(ctx.Bobfile.DockerImages) == 0 {
		return registryCredentials.OkWithReason("n/a")
	}

	if isEnvVarPresent("DOCKER_CREDS") {
		return registryCredentials.Ok()
	}

	return registryCredentials.Fail("DOCKER_CREDS not defined")
}

func passableEnvVarsPresent(ctx *CheckContext) error {
	keyVisitedChecker := map[string]bool{}

	for _, builder := range ctx.Bobfile.Builders {
		for _, envKey := range builder.PassEnvs {
			if _, set := keyVisitedChecker[envKey]; set {
				continue // ENV already visited
			}

			keyVisitedChecker[envKey] = true

			check := ctx.NewCheck(fmt.Sprintf("ENV(%s)", envKey))

			if isEnvVarPresent(envKey) {
				check.Ok()
			} else {
				check.Fail("Not set")
			}
		}
	}

	return nil
}

// plumbing below

type CheckResult struct {
	Name   string
	Ok     bool
	Reason string
}

type CheckContext struct {
	Bobfile *Bobfile
	Results []CheckResult
}

type CheckResultBuilder struct {
	name string
	ctx  *CheckContext
}

func (c *CheckContext) NewCheck(name string) CheckResultBuilder {
	return CheckResultBuilder{name: name, ctx: c}
}

func (c *CheckResultBuilder) Ok() error {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name: c.name,
		Ok:   true,
	})

	return nil
}

func (c *CheckResultBuilder) OkWithReason(reason string) error {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name:   c.name,
		Ok:     true,
		Reason: reason,
	})

	return nil
}

func (c *CheckResultBuilder) Fail(reason string) error {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name:   c.name,
		Ok:     false,
		Reason: reason,
	})

	return nil
}
