package openshift

import (
	"fmt"
	"github.com/michaelsauter/ocdiff/utils"
	"strings"
)

var availableKinds = []string{
	"svc",
	"route",
	"dc",
	"bc",
	"is",
	"pvc",
	"template",
	"cm",
	"secret",
	"rolebinding",
	"serviceaccount",
}

type ResourceFilter struct {
	Kinds []string
	Name  string
	Label string
}

func (f *ResourceFilter) String() string {
	return fmt.Sprintf("Kind: %s, Name: %s, Label: %s", f.Kinds, f.Name, f.Label)
}

func (f *ResourceFilter) SatisfiedBy(item *ResourceItem) bool {
	if len(f.Name) > 0 && f.Name != item.FullName() {
		return false
	}

	if len(f.Kinds) > 0 && !utils.Includes(f.Kinds, item.Kind) {
		return false
	}

	if len(f.Label) > 0 {
		labelParts := strings.Split(f.Label, "=")
		if _, ok := item.Labels[labelParts[0]]; !ok {
			return false
		} else if item.Labels[labelParts[0]].(string) != labelParts[1] {
			return false
		}
	}

	return true
}

func (f *ResourceFilter) ConvertToTarget() string {
	if len(f.Name) > 0 {
		nameParts := strings.Split(f.Name, "/")
		return nameParts[0]
	}
	kinds := f.Kinds
	if len(kinds) == 0 {
		kinds = availableKinds
	}
	return strings.Join(kinds, ",")
}
