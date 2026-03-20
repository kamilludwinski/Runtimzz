package versionutils

import (
	"sort"
	"strconv"
	"strings"
)

// SortVersions sorts semversions, major -> minor -> patch
func SortVersions(versions []string, desc bool) {
	sort.Slice(versions, func(i, j int) bool {
		vi := parseVersion(versions[i])
		vj := parseVersion(versions[j])

		for k := range 3 {
			if vi[k] != vj[k] {
				if desc {
					return vi[k] > vj[k]
				}
				return vi[k] < vj[k]
			}
		}
		return false
	})
}

func parseVersion(v string) [3]int {
	parts := strings.Split(v, ".")
	var res [3]int

	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err == nil {
			res[i] = n
		}
	}

	return res
}

// CompareVersions compares two x.y.z version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareVersions(a, b string) int {
	va := parseVersion(a)
	vb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] < vb[i] {
			return -1
		}
		if va[i] > vb[i] {
			return 1
		}
	}
	return 0
}

// ResolveVersion resolves an input version string against a list of candidate versions.
// Supported forms:
//   - "latest": returns the highest candidate version.
//   - "x": highest version matching major x (x.y.z).
//   - "x.y": highest version matching major x and minor y (x.y.z).
//   - "x.y.z": exact match only.
//
// It returns the resolved version and true if a match was found, or ("", false) otherwise.
func ResolveVersion(input string, candidates []string) (string, bool) {
	in := strings.TrimSpace(input)
	if len(candidates) == 0 || in == "" {
		return "", false
	}

	// Special keyword: latest -> highest candidate.
	if strings.EqualFold(in, "latest") {
		versions := append([]string(nil), candidates...)
		SortVersions(versions, true)
		return versions[0], true
	}

	parts := strings.Split(in, ".")
	if len(parts) > 3 {
		return "", false
	}

	var seg [3]int
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			// malformed input like "abc" -> no match
			return "", false
		}
		seg[i] = n
	}

	// Exact form x.y.z: require an exact candidate match.
	if len(parts) == 3 {
		for _, c := range candidates {
			if c == in {
				return c, true
			}
		}
		return "", false
	}

	// Prefix forms x or x.y: filter by matching major (and minor, if provided),
	// then return the highest version from the filtered set.
	var filtered []string
	for _, c := range candidates {
		cv := parseVersion(c)
		if cv[0] != seg[0] {
			continue
		}
		if len(parts) == 2 && cv[1] != seg[1] {
			continue
		}
		filtered = append(filtered, c)
	}

	if len(filtered) == 0 {
		return "", false
	}

	SortVersions(filtered, true)
	return filtered[0], true
}
