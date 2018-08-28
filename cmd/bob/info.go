package main

import (
	"fmt"
	"os"
)

func info(bobfile *Bobfile) error {
	metadata, errMetadata := resolveMetadataFromVersionControl()
	if errMetadata != nil {
		return errMetadata
	}

	fmt.Printf("Project name: %s\n", bobfile.ProjectName)
	fmt.Printf("VcKind: %s\n", metadata.VcKind)
	fmt.Printf("Revision ID (full): %s (%s)\n", metadata.RevisionIdShort, metadata.RevisionId)
	fmt.Printf("Friendly revision: %s\n", metadata.FriendlyRevisionId)

	for _, builder := range bobfile.Builders {
		fmt.Printf("\n----\nBuilder: %s\n", builder.Name)
		fmt.Printf("Mount dir: %s\n", builder.MountDestination)
		for _, envKey := range builder.PassEnvs {
			setOrNot := "NOT SET"
			if os.Getenv(envKey) != "" {
				setOrNot = "SET"
			}

			fmt.Printf("ENV: %s (%s)\n", envKey, setOrNot)
		}
	}

	return nil
}
