package main

import (
	"fmt"

	"github.com/function61/gokit/os/osutil"
)

func RunChecks(buildCtx *BuildContext) ([]CheckResult, error) {
	ctx := &CheckContext{
		BuildContext: buildCtx,
		Results:      []CheckResult{},
	}

	checks := []func(*CheckContext) error{
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

	exists, errChecking := osutil.Exists("LICENSE")
	if errChecking != nil {
		return errChecking
	}

	if exists {
		licenseCheck.Ok()
	} else {
		licenseCheck.Fail("Project must have a LICENSE file")
	}

	return nil
}

func readmePresent(ctx *CheckContext) error {
	readmeCheck := ctx.NewCheck("Readme present")

	exists, errChecking := osutil.Exists("README.md")
	if errChecking != nil {
		return errChecking
	}

	if exists {
		readmeCheck.Ok()
	} else {
		readmeCheck.Fail("Project must have a README.md file")
	}

	return nil
}

func passableEnvVarsPresent(ctx *CheckContext) error {
	keyVisitedChecker := map[string]bool{}

	for _, builder := range ctx.BuildContext.Bobfile.Builders {
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
	BuildContext *BuildContext
	Results      []CheckResult
}

type CheckResultBuilder struct {
	name string
	ctx  *CheckContext
}

func (c *CheckContext) NewCheck(name string) CheckResultBuilder {
	return CheckResultBuilder{name: name, ctx: c}
}

func (c *CheckResultBuilder) Ok() {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name: c.name,
		Ok:   true,
	})
}

func (c *CheckResultBuilder) OkWithReason(reason string) {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name:   c.name,
		Ok:     true,
		Reason: reason,
	})
}

func (c *CheckResultBuilder) Fail(reason string) {
	c.ctx.Results = append(c.ctx.Results, CheckResult{
		Name:   c.name,
		Ok:     false,
		Reason: reason,
	})
}
