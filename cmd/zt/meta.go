package main

import (
	"fmt"
	"io/fs"
	"sort"

	"github.com/spf13/cobra"

	tools "github.com/zeithrold/tools"
)

func newVersionCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the zt version",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(a.stdout, tools.Version())
			return err
		},
	}
}

func newLicenseCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "license",
		Short: "Print every bundled license and dependency notice",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := licenseFileNames()
			if err != nil {
				return err
			}
			for _, name := range names {
				data, err := tools.LicenseFiles.ReadFile(name)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(a.stdout, "%s\n%s\n", name, data); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// licenseFileNames returns every embedded notice in stable path order, so a
// newly bundled dependency notice is printed without editing this command.
func licenseFileNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(tools.LicenseFiles, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		names = append(names, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
