package version

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewConstraint(t *testing.T) {
	cases := []struct {
		input string
		ors   int
		count int
		err   bool
	}{
		{">= 1.2", 1, 1, false},
		{"1.0", 1, 1, false},
		{">= 1.x", 0, 0, true},
		{">= 1.2, < 1.0", 1, 2, false},
		{">= 1.2, < 1.0 || ~> 2.0, < 3", 2, 2, false},

		// Out of bounds
		{"11387778780781445675529500000000000000000", 0, 0, true},
	}

	for _, tc := range cases {
		v, err := NewConstraint(tc.input)
		if tc.err && err == nil {
			t.Fatalf("expected error for input: %s", tc.input)
		} else if !tc.err && err != nil {
			t.Fatalf("error for input %s: %s", tc.input, err)
		}
		if tc.err {
			continue
		}

		actual := len(v)
		if actual != tc.ors {
			t.Fatalf("input: %s\nexpected ors: %d\nactual: %d",
				tc.input, tc.ors, actual)
		}

		actual = len(v[0])
		if actual != tc.count {
			t.Fatalf("input: %s\nexpected len: %d\nactual: %d",
				tc.input, tc.count, actual)
		}
	}
}

func TestConstraintCheck(t *testing.T) {
	cases := []struct {
		constraint string
		version    string
		check      bool
	}{
		{">= 1.0, < 1.2 || > 1.3", "1.1.5", true},
		{">= 1.0, < 1.2 || > 1.3", "1.3.2", true},
		{">= 1.0, < 1.2 || > 1.3", "1.2.3", false},
		{">= 1.0, < 1.2", "1.1.5", true},
		{"< 1.0, < 1.2", "1.1.5", false},
		{"= 1.0", "1.1.5", false},
		{"= 1.0", "1.0.0", true},
		{"1.0", "1.0.0", true},
		{"~> 1.0", "2.0", false},
		{"~> 1.0", "1.1", true},
		{"~> 1.0", "1.2.3", true},
		{"~> 1.0.0", "1.2.3", false},
		{"~> 1.0.0", "1.0.7", true},
		{"~> 1.0.0", "1.1.0", false},
		{"~> 1.0.7", "1.0.4", false},
		{"~> 1.0.7", "1.0.7", true},
		{"~> 1.0.7", "1.0.8", true},
		{"~> 1.0.7", "1.0.7.5", true},
		{"~> 1.0.7", "1.0.6.99", false},
		{"~> 1.0.7", "1.0.8.0", true},
		{"~> 1.0.9.5", "1.0.9.5", true},
		{"~> 1.0.9.5", "1.0.9.4", false},
		{"~> 1.0.9.5", "1.0.9.6", true},
		{"~> 1.0.9.5", "1.0.9.5.0", true},
		{"~> 1.0.9.5", "1.0.9.5.1", true},
		{"~> 2.0", "2.1.0-beta", true}, // https://semver.org/#spec-item-11 (#3)
		{"~> 2.0", "2.0.0-beta", false},
		{"~> 2.1.0-a", "2.2.0", false},
		{"~> 2.1.0-a", "2.1.0", false},
		{"~> 2.1.0-a", "2.1.0-beta", true},
		{"~> 2.1.0-a", "2.2.0-alpha", false},
		{"> 2.0", "2.1.0-beta", true}, // https://semver.org/#spec-item-11 (#3)
		{"> 2.0", "2.0.0-beta", false},
		{">= 2.0", "2.0.0-beta", false},
		{">= 2.1.0-a", "2.1.0-beta", true},
		{">= 2.1.0-a", "2.1.1-beta", true}, // segment comparison takes precedence over prerelease comparison, but is still valid to compare
		{">= 2.0.0", "2.1.0-beta", true},   // https://semver.org/#spec-item-11 (#3)
		{">= 2.0.0", "2.0.0-beta", false},
		{">= 2.1.0-a", "2.1.1", true},
		{">= 2.1.0-a", "2.1.1-beta", true}, // segment comparison takes precedence over prerelease comparison, but is still valid to compare
		{">= 2.1.0-a", "2.1.0", true},
		{"<= 2.1.0-a", "2.0.0", true},
		{"^1.1", "1.1.1", true},
		{"^1.1", "1.2.3", true},
		{"^1.1", "2.1.0", false},
		{"^1.1.2", "1.1.1", false},
		{"^1.1.2", "1.1.2", true},
		{"^1.1.2", "1.1.2.3", true},
		{"~1", "1.3.5", true},
		{"~1", "2.1.0", false},
		{"~1.1", "1.1.1", true},
		{"~1.1", "1.2.3", false},
		{"~1.1.2", "1.1.1", false},
		{"~1.1.2", "1.1.2", true},
		{"~1.1.2", "1.1.2.3", true},
		{"> 1.0.0-alpha", "1.0.0-alpha.1", true},
		{"> 1.0.0-alpha.1", "1.0.0-alpha.beta", true},
		{"> 1.0.0-alpha.beta", "1.0.0-beta", true},
		{"> 1.0.0-beta", "1.0.0-beta.2", true},
		{"> 1.0.0-beta.2", "1.0.0-beta.11", true},
		{"> 1.0.0-beta.11", "1.0.0-rc.1", true},
		{"> 1.0.0-rc.1", "1.0.0", true},
		{"< 1.0.0-alpha", "1.0.0-alpha.1", false},
		{"< 1.0.0-alpha.1", "1.0.0-alpha.beta", false},
		{"< 1.0.0-alpha.beta", "1.0.0-beta", false},
		{"< 1.0.0-beta", "1.0.0-beta.2", false},
		{"< 1.0.0-beta.2", "1.0.0-beta.11", false},
		{"< 1.0.0-beta.11", "1.0.0-rc.1", false},
		{"< 1.0.0-rc.1", "1.0.0", false},
		{"< 1.0.0", "1.0.0-rc.1", true},
		{"> 1.0.0", "1.0.0-rc.1", false},
		{"< 0.9.12-r1", "0.9.9-r0", true},  // regression
		{"< 0.9.9-r1", "0.9.9-r0", true},   // regression
		{"< 0.9.9-r1", "0.9.9-r11", false}, // regression
		{"> 0.9.9-r1", "0.9.9-r11", true},  // regression
		{"<= 1.3.3-r0", "1.3.2-r0", true},  // introduced for removal of prereleaseCheck from <=
		//Tests below to test == versions with different pre releases for each operator
		{"<= 1.3.3-r0", "1.3.3-r0", true},
		{"<= 1.3.3-r0", "1.3.3-r1", false},
		{"<= 1.3.3-r1", "1.3.3-r0", true},
		{"= 1.3.3-r1", "1.3.3-r1", true},
		{"= 1.3.3-r1", "1.3.3-r0", false},
		{"> 1.3.3-r1", "1.3.3-r0", false},
		{"> 1.3.3-r1", "1.3.3-r3", true},
		{">= 1.3.3-r1", "1.3.3-r0", false},
		{">= 1.3.3-r1", "1.3.3-r5", true},
		{"< 1.3.3-r1", "1.3.3-r0", true},
		{"< 1.3.3-r1", "1.3.3-r5", false},
	}

	for _, tc := range cases {
		c, err := NewConstraint(tc.constraint)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		v, err := NewVersion(tc.version)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := c.Check(v)
		expected := tc.check
		if actual != expected {
			t.Errorf("\nVersion: %s\nConstraint: %s\nExpected: %#v",
				tc.version, tc.constraint, expected)
		}
	}
}

func TestConstraintsString(t *testing.T) {
	cases := []struct {
		constraint string
		result     string
	}{
		{">= 1.0, < 1.2", ""},
		{"~> 1.0.7", ""},
		{">= 1.0, < 1.2 || > 1.3", ""},
	}

	for _, tc := range cases {
		c, err := NewConstraint(tc.constraint)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := c.String()
		expected := tc.result
		if expected == "" {
			expected = tc.constraint
		}

		if actual != expected {
			t.Fatalf("Constraint: %s\nExpected: %#v\nActual: %s",
				tc.constraint, expected, actual)
		}
	}
}

func TestConstraintsJson(t *testing.T) {
	type MyStruct struct {
		MustVer Constraints
	}
	var (
		vc  MyStruct
		err error
	)
	jsBytes := []byte(`{"MustVer":"=1.2, =1.3"}`)
	// data -> struct
	err = json.Unmarshal(jsBytes, &vc)
	if err != nil {
		t.Fatalf("expected: json.Unmarshal to succeed\nactual: failed with error %v", err)
	}
	// struct -> data
	data, err := json.Marshal(&vc)
	if err != nil {
		t.Fatalf("expected: json.Marshal to succeed\nactual: failed with error %v", err)
	}

	if !bytes.Equal(data, jsBytes) {
		t.Fatalf("expected: %s\nactual: %s", jsBytes, data)
	}
}
