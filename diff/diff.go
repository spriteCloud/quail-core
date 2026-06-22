// Package diff parses a unified diff into per-file added line ranges.
//
// The output is intentionally minimal: we only care about which lines were
// ADDED in the PR (so we can intersect with AST symbol spans). Context lines
// and deletions are tracked just well enough to keep line numbers correct.
package diff

import (
	"bufio"
	"strconv"
	"strings"
)

type Range struct{ Start, End int } // inclusive, 1-based, in the NEW file

type File struct {
	Path    string
	OldPath string
	Added   []Range
	NewBlob string // empty unless caller fills it later
	OldBlob string // pre-PR contents; gh package populates from the API
	Status  string // "added", "modified", "removed", "renamed"
}

func Parse(unified string) []File {
	var files []File
	var cur *File
	sc := bufio.NewScanner(strings.NewReader(unified))
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	var newLine int
	var inHunk bool
	flush := func() {
		if cur != nil {
			files = append(files, *cur)
		}
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			cur = &File{Status: "modified"}
			inHunk = false
		case strings.HasPrefix(line, "--- "):
			if cur == nil {
				continue
			}
			p := strings.TrimPrefix(line, "--- ")
			if p == "/dev/null" {
				cur.Status = "added"
			} else if strings.HasPrefix(p, "a/") {
				cur.OldPath = p[2:]
			} else {
				cur.OldPath = p
			}
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				continue
			}
			p := strings.TrimPrefix(line, "+++ ")
			if p == "/dev/null" {
				cur.Status = "removed"
			} else if strings.HasPrefix(p, "b/") {
				cur.Path = p[2:]
			} else {
				cur.Path = p
			}
			if cur.OldPath != "" && cur.Path != "" && cur.OldPath != cur.Path && cur.Status == "modified" {
				cur.Status = "renamed"
			}
		case strings.HasPrefix(line, "@@"):
			if cur == nil {
				continue
			}
			inHunk = true
			newLine = parseHunkNewStart(line)
		case inHunk && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			if cur == nil {
				continue
			}
			cur.Added = mergeRange(cur.Added, newLine)
			newLine++
		case inHunk && strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// deletion — do not advance newLine
		case inHunk:
			newLine++
		}
	}
	flush()
	return files
}

func parseHunkNewStart(header string) int {
	// "@@ -a,b +c,d @@ ..." — return c
	plus := strings.Index(header, "+")
	if plus == -1 {
		return 1
	}
	rest := header[plus+1:]
	end := strings.IndexAny(rest, " ,")
	if end == -1 {
		end = len(rest)
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func mergeRange(rs []Range, line int) []Range {
	if len(rs) == 0 {
		return []Range{{Start: line, End: line}}
	}
	last := &rs[len(rs)-1]
	if line == last.End+1 {
		last.End = line
		return rs
	}
	return append(rs, Range{Start: line, End: line})
}

// Intersects reports whether [start,end] (inclusive) overlaps any added range.
func Intersects(added []Range, start, end int) bool {
	for _, r := range added {
		if start <= r.End && r.Start <= end {
			return true
		}
	}
	return false
}
