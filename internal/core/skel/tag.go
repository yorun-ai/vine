package skel

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// HasTagFlag reports whether a comma-separated skel tag contains an exact flag.
func HasTagFlag(tag reflect.StructTag, name string) bool {
	for option := range strings.SplitSeq(tag.Get("skel"), ",") {
		if strings.TrimSpace(option) == name {
			return true
		}
	}
	return false
}

// TagIndex reads an index(n) attribute. Invalid or repeated indexes are errors;
// unrelated attributes are ignored. The caller validates the argument range.
func TagIndex(tag reflect.StructTag) (index int, found bool, err error) {
	for option := range strings.SplitSeq(tag.Get("skel"), ",") {
		option = strings.TrimSpace(option)
		if option != "index" && !strings.HasPrefix(option, "index(") {
			continue
		}
		if found {
			return 0, false, fmt.Errorf("duplicate skel index attribute")
		}
		argument, ok := strings.CutPrefix(option, "index(")
		if !ok || !strings.HasSuffix(argument, ")") {
			return 0, false, fmt.Errorf("invalid skel index attribute %q", option)
		}
		index, err = strconv.Atoi(strings.TrimSuffix(argument, ")"))
		if err != nil {
			return 0, false, fmt.Errorf("invalid skel index attribute %q: %w", option, err)
		}
		found = true
	}
	return index, found, nil
}
