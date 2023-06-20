package selfupdate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
}

func (v *Version) Greater(target *Version) bool {
	return v.Compare(target) < 0
}

func (v *Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v *Version) Compare(o *Version) int {
	if d := compareSegment(v.Major, o.Major); d != 0 {
		return d
	}
	if d := compareSegment(v.Minor, o.Minor); d != 0 {
		return d
	}
	if d := compareSegment(v.Patch, o.Patch); d != 0 {
		return d
	}
	return 0
}

func ParseVersion(v string) (*Version, error) {
	m := versionRegex.FindStringSubmatch(v)
	if m == nil {
		return nil, errors.New("ErrInvalidSemver")
	}

	out := &Version{
		Major: 0,
		Minor: 0,
		Patch: 0,
	}

	var err error
	out.Major, err = strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing version segment: %s", err)
	}

	if m[2] != "" {
		out.Minor, err = strconv.ParseUint(strings.TrimPrefix(m[2], "."), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing version segment: %s", err)
		}
	} else {
		out.Minor = 0
	}

	if m[3] != "" {
		out.Patch, err = strconv.ParseUint(strings.TrimPrefix(m[3], "."), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing version segment: %s", err)
		}
	} else {
		out.Patch = 0
	}
	return out, nil
}

func compareSegment(v, o uint64) int {
	if v < o {
		return -1
	}
	if v > o {
		return 1
	}

	return 0
}
