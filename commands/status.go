package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
)

// Status prints the drift between desired and current state to STDOUT.
func Status(compareOptions *cli.CompareOptions) (bool, *openshift.Changeset, error) {
	return calculateChangeset(compareOptions)
}

func calculateChangeset(compareOptions *cli.CompareOptions) (bool, *openshift.Changeset, error) {
	updateRequired := false

	where := strings.Join(compareOptions.TemplateDirs, ", ")
	if len(compareOptions.TemplateDirs) == 1 && compareOptions.TemplateDirs[0] == "." {
		where, _ = os.Getwd()
	}

	fmt.Printf(
		"Comparing templates in %s with OCP namespace %s.\n",
		where,
		compareOptions.Namespace,
	)

	if len(compareOptions.Resource) > 0 && len(compareOptions.Selector) > 0 {
		fmt.Printf(
			"Limiting resources to %s with selector %s.\n",
			compareOptions.Resource,
			compareOptions.Selector,
		)
	} else if len(compareOptions.Selector) > 0 {
		fmt.Printf(
			"Limiting to resources with selector %s.\n",
			compareOptions.Selector,
		)
	} else if len(compareOptions.Resource) > 0 {
		fmt.Printf(
			"Limiting resources to %s.\n",
			compareOptions.Resource,
		)
	}

	resource := compareOptions.Resource

	filter, err := openshift.NewResourceFilter(resource, compareOptions.Selector, compareOptions.Exclude)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	templateBasedList, err := assembleTemplateBasedResourceList(
		filter,
		compareOptions,
	)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	platformBasedList, err := assemblePlatformBasedResourceList(filter, compareOptions)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	platformResourcesWord := "resources"
	if platformBasedList.Length() == 1 {
		platformResourcesWord = "resource"
	}
	templateResourcesWord := "resources"
	if templateBasedList.Length() == 1 {
		templateResourcesWord = "resource"
	}
	fmt.Printf(
		"Found %d %s in OCP cluster (current state) and %d %s in processed templates (desired state).\n\n",
		platformBasedList.Length(),
		platformResourcesWord,
		templateBasedList.Length(),
		templateResourcesWord,
	)

	if templateBasedList.Length() == 0 && !compareOptions.Force {
		fmt.Printf("No items where found in desired state. ")
		if len(compareOptions.Resource) == 0 && len(compareOptions.Selector) == 0 {
			fmt.Printf(
				"Are there any templates in %s?\n",
				where,
			)
		} else {
			fmt.Printf(
				"Possible reasons are:\n"+
					"* No templates are located in %s\n",
				where,
			)
			if len(compareOptions.Resource) > 0 {
				fmt.Printf(
					"* No templates contain resources of kinds: %s\n",
					compareOptions.Resource,
				)
			}
			if len(compareOptions.Selector) > 0 {
				fmt.Printf(
					"* No templates contain resources matching selector: %s\n",
					compareOptions.Selector,
				)
			}
		}
		fmt.Println("\nRefusing to continue without --force")
		return updateRequired, &openshift.Changeset{}, nil
	}

	changeset, err := compare(
		platformBasedList,
		templateBasedList,
		compareOptions.UpsertOnly,
		compareOptions.Diff,
		compareOptions.IgnorePaths,
	)
	if err != nil {
		return false, changeset, err
	}
	updateRequired = !changeset.Blank()
	return updateRequired, changeset, nil
}

func compare(remoteResourceList *openshift.ResourceList, localResourceList *openshift.ResourceList, upsertOnly bool, diff string, ignorePaths []string) (*openshift.Changeset, error) {
	changeset, err := openshift.NewChangeset(remoteResourceList, localResourceList, upsertOnly, ignorePaths)
	if err != nil {
		return changeset, err
	}

	for _, change := range changeset.Noop {
		fmt.Printf("* %s is in sync\n", change.ItemName())
	}

	for _, change := range changeset.Delete {
		cli.PrintRedf("- %s to delete\n", change.ItemName())
		fmt.Print(change.Diff())
	}

	for _, change := range changeset.Create {
		cli.PrintGreenf("+ %s to create\n", change.ItemName())
		fmt.Print(change.Diff())
	}

	for _, change := range changeset.Update {
		cli.PrintYellowf("~ %s to update\n", change.ItemName())
		if diff == "text" {
			fmt.Print(change.Diff())
		} else {
			fmt.Println(change.JsonPatches(true))
		}
	}

	fmt.Printf("\nSummary: %d in sync, ", len(changeset.Noop))
	cli.PrintGreenf("%d to create", len(changeset.Create))
	fmt.Printf(", ")
	cli.PrintYellowf("%d to update", len(changeset.Update))
	fmt.Printf(", ")
	cli.PrintRedf("%d to delete\n\n", len(changeset.Delete))

	return changeset, nil
}

func assembleTemplateBasedResourceList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions) (*openshift.ResourceList, error) {
	var inputs [][]byte

	// read files in folders and assemble lists for kinds
	for i, templateDir := range compareOptions.TemplateDirs {
		files, err := ioutil.ReadDir(templateDir)
		if err != nil {
			return nil, err
		}
		filePattern := ".*\\.ya?ml$"
		re := regexp.MustCompile(filePattern)
		for _, file := range files {
			matched := re.MatchString(file.Name())
			if !matched {
				continue
			}
			cli.DebugMsg("Reading template", file.Name())
			processedOut, err := openshift.ProcessTemplate(templateDir, file.Name(), compareOptions.ParamDirs[i], compareOptions)
			if err != nil {
				return nil, fmt.Errorf("Could not process %s template: %s", file.Name(), err)
			}
			inputs = append(inputs, processedOut)
		}
	}

	return openshift.NewTemplateBasedResourceList(filter, inputs...)
}

func assemblePlatformBasedResourceList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions) (*openshift.ResourceList, error) {
	exportedOut, err := openshift.ExportResources(filter, compareOptions.Namespace)
	if err != nil {
		return nil, fmt.Errorf("Could not export %s resources", filter.String())
	}
	return openshift.NewPlatformBasedResourceList(filter, exportedOut)
}
