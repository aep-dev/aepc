package writer_utils

import (
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
