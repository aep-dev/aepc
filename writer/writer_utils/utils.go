package writer_utils

import (
	"fmt"
	"strings"

	"github.com/aep-dev/aepc/parser"
)

// return the collection name of the resource, but deduplicate
// the name of the previous parent
// e.g:
// - book-editions becomes editions under the parent resource book.
func CollectionName(r *parser.ParsedResource) string {
	collectionName := r.Plural
	if len(r.ParsedParents) > 0 {
		parent := r.ParsedParents[0].Kind
		// if collectionName has a prefix of parent, remove it
		if strings.HasPrefix(collectionName, parent) {
			collectionName = strings.TrimPrefix(collectionName, parent+"-")
		}
	}
	return collectionName
}

// GeneratePatternStrings generates the pattern strings for a resource
// TODO(yft): support multiple parents
func GeneratePatternStrings(r *parser.ParsedResource) []string {

	// Base pattern without params
	pattern := fmt.Sprintf("%v/{%v}", CollectionName(r), r.Kind)
	if len(r.ParsedParents) > 0 {
		parentParts := GeneratePatternStrings(r.ParsedParents[0])
		pattern = fmt.Sprintf("%v/%v", parentParts[0], pattern)
	}
	return []string{pattern}
}
