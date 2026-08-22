package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Compare returns -1, 0, or 1 for two semantic versions. A leading v is ignored.
// Invalid versions are compared deterministically as strings after normalization.
func Compare(a, b string) int {
	aa, oka := parse(a)
	bb, okb := parse(b)
	if !oka || !okb {
		a = strings.TrimPrefix(a, "v")
		b = strings.TrimPrefix(b, "v")
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	for i := 0; i < 3; i++ {
		if aa.n[i] < bb.n[i] {
			return -1
		}
		if aa.n[i] > bb.n[i] {
			return 1
		}
	}
	if aa.pre == bb.pre {
		return 0
	}
	if aa.pre == "" {
		return 1
	}
	if bb.pre == "" {
		return -1
	}
	if aa.pre < bb.pre {
		return -1
	}
	return 1
}

// Satisfies reports whether v matches a simple constraint expression.
// Supported operators are >=, <=, >, <, =, ^ and ~. Whitespace-separated
// expressions are treated as an AND range.
func Satisfies(v, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	for _, token := range strings.Fields(constraint) {
		op := "="
		for _, candidate := range []string{"^", "~", ">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				op = candidate
				token = strings.TrimPrefix(token, candidate)
				break
			}
		}
		cmp := Compare(v, token)
		switch op {
		case "=":
			if cmp != 0 {
				return false
			}
		case ">":
			if cmp <= 0 {
				return false
			}
		case ">=":
			if cmp < 0 {
				return false
			}
		case "<":
			if cmp >= 0 {
				return false
			}
		case "<=":
			if cmp > 0 {
				return false
			}
		case "^":
			base, ok := parse(token)
			if !ok || cmp < 0 {
				return false
			}
			if base.n[0] > 0 && Compare(v, fmt.Sprintf("%d.0.0", base.n[0]+1)) >= 0 {
				return false
			}
			if base.n[0] == 0 && Compare(v, fmt.Sprintf("0.%d.0", base.n[1]+1)) >= 0 {
				return false
			}
		case "~":
			base, ok := parse(token)
			if !ok || cmp < 0 || Compare(v, fmt.Sprintf("%d.%d.0", base.n[0], base.n[1]+1)) >= 0 {
				return false
			}
		}
	}
	return true
}

type parsed struct {
	n   [3]int
	pre string
}

func parse(v string) (parsed, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, "-", 2)
	nums := strings.Split(parts[0], ".")
	if len(nums) != 3 {
		return parsed{}, false
	}
	var p parsed
	for i := range nums {
		n, err := strconv.Atoi(nums[i])
		if err != nil || n < 0 {
			return parsed{}, false
		}
		p.n[i] = n
	}
	if len(parts) == 2 {
		p.pre = parts[1]
	}
	return p, true
}
