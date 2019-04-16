// Copyright 2019 CanonicalLtd

package call

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	attributePlaceholder = regexp.MustCompile("\\{(.*?)\\}")
)

// Attributes represents a collection of attributes.
type Attributes map[string]interface{}

func (a *Attributes) templateValue(match string) string {
	attributeName := strings.Trim(match, "{}")
	value, ok := (*a)[attributeName]
	if !ok {
		return match
	}
	return fmt.Sprintf("%v", value)
}

func (a *Attributes) renderString(url string) string {
	return attributePlaceholder.ReplaceAllStringFunc(url, a.templateValue)
}
