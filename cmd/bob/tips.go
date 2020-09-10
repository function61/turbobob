package main

import (
	"fmt"

	"github.com/function61/gokit/os/osutil"
	"github.com/scylladb/termtables"
	"github.com/spf13/cobra"
)

func tipsEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "tips",
		Short: "Show useful commands & pro-tips",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(tips())
		},
	}
}

func tips() error {
	bobfile, err := readBobfile()
	if err != nil {
		return err
	}

	shimConf, err := readShimConfig()
	if err != nil {
		return err
	}

	builder, err := findBuilder(bobfile, shimConf.BuilderName)
	if err != nil {
		return err
	}

	baseImgConf, err := loadBaseImageConf()
	if err != nil {
		return err
	}

	tipsTbl := termtables.CreateTable()
	tipsTbl.AddHeaders(" Source", "")

	row := func(label string, value string) {
		tipsTbl.AddRow(" "+label, value)
	}

	importantRow := func(label string, value string) {
		tipsTbl.AddRow("*"+label, value)
	}

	for _, command := range builder.DevShellCommands {
		if command.Important {
			importantRow("Dev command", command.Command)
		} else {
			row("Dev command", command.Command)
		}
	}

	for _, command := range baseImgConf.DevShellCommands {
		if command.Important {
			importantRow("Base image", command.Command)
		} else {
			row("Base image", command.Command)
		}
	}

	for _, proTip := range builder.DevProTips {
		importantRow("Pro tip", proTip)
	}

	for _, tip := range shimConf.DynamicProTipsFromHost {
		importantRow("Pro tip", tip)
	}

	fmt.Println("USEFUL COMMANDS, PRO-TIPS")
	fmt.Print(tipsTbl.Render())
	fmt.Println("* = Show on entering build container ($ bob dev)")

	return nil
}
