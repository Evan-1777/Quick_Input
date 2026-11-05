package app

import (
	"strings"

	"quick-input/internal/store"
)

// Filter scopes, matching the store's SnippetFilter contract.
const (
	scopeAll       = "all"
	scopeUngrouped = "ungrouped"
	scopeGroup     = "group"
)

// FilterState is the in-memory filter shared by the overlay and the manager.
type FilterState struct {
	Keyword string
	Scope   string
	GroupID *int64
	TagIDs  []int64
}

// filterSnippets keeps the old frontend's semantics: a trimmed, lower-cased
// keyword matched against title and content with a newline between, one
// group scope, and the intersection of every chosen tag.
func filterSnippets(snippets []store.Snippet, f FilterState) []store.Snippet {
	keyword := strings.ToLower(strings.TrimSpace(f.Keyword))
	result := make([]store.Snippet, 0, len(snippets))
	for _, snippet := range snippets {
		if keyword != "" && !strings.Contains(strings.ToLower(snippet.Title+"\n"+snippet.Content), keyword) {
			continue
		}
		switch f.Scope {
		case scopeUngrouped:
			if snippet.GroupID != nil {
				continue
			}
		case scopeGroup:
			if f.GroupID == nil || snippet.GroupID == nil || *snippet.GroupID != *f.GroupID {
				continue
			}
		}
		if !hasEveryTag(snippet.Tags, f.TagIDs) {
			continue
		}
		result = append(result, snippet)
	}
	return result
}

// nextIndex wraps a list index in either direction.
func nextIndex(current, total, step int) int {
	if total <= 0 {
		return 0
	}
	if current < 0 || current >= total {
		return 0
	}
	return (current + step + total) % total
}

// toggleID adds or removes one id.
func toggleID(ids []int64, id int64) []int64 {
	for i, existing := range ids {
		if existing == id {
			result := make([]int64, 0, len(ids)-1)
			result = append(result, ids[:i]...)
			return append(result, ids[i+1:]...)
		}
	}
	return append(ids, id)
}

func hasID(ids []int64, id int64) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func hasEveryTag(tags []store.Tag, ids []int64) bool {
	for _, id := range ids {
		found := false
		for _, tag := range tags {
			if tag.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// pruneFilter drops a group or tag that no longer exists, so a stale filter
// cannot hide every row without an obvious cause.
func pruneFilter(f *FilterState, groups []store.Group, tags []store.Tag) {
	if f.Scope == scopeGroup {
		valid := false
		if f.GroupID != nil {
			for _, group := range groups {
				if group.ID == *f.GroupID {
					valid = true
					break
				}
			}
		}
		if !valid {
			f.Scope, f.GroupID = scopeAll, nil
		}
	} else {
		f.GroupID = nil
	}
	if f.Scope == "" {
		f.Scope = scopeAll
	}
	if len(f.TagIDs) > 0 {
		kept := f.TagIDs[:0]
		for _, id := range f.TagIDs {
			for _, tag := range tags {
				if tag.ID == id {
					kept = append(kept, id)
					break
				}
			}
		}
		f.TagIDs = kept
	}
}
