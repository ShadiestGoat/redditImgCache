package main_test

import (
	"os"
	"strings"
	"testing"
	"github.com/shadiestgoat/redditImgCache"
)

func testConfig(t *testing.T, s, exp string, env map[string]string) {
	// reset env
	prev := os.Environ()
	for _, v := range prev {
		os.Unsetenv(strings.SplitN(v, "=", 2)[0])
	}

	for k, v := range env {
		os.Setenv(k, v)
	}

	old := s
	main.LoadEnvConfig(&s)

	if s != exp {
		t.Log(os.Environ())
		t.Errorf("Unexpected config parse: og: '%v', exp: '%v', got: '%v'", old, exp, s)
	}
}

func TestLoadEnvConfig(t *testing.T) {
	t.Run("correct_vars", func(t *testing.T) {
		testConfig(
			t,
			`$REAL_VAR:$FAKE_VAR:$1BAD_VAR:$REAL_VAR_`,
			`real:$FAKE_VAR:$1BAD_VAR:real_`,
			map[string]string{
				"REAL_VAR": "real",
			},
		)
	})

	t.Run("recursive", func(t *testing.T) {
		testConfig(
			t,
			`$$REC`,
			`$COOL`,
			map[string]string{
				"REC": "COOL",
				"COOL": "not-so-cool",
			},
		)
		testConfig(
			t,
			`$AB$CD`,
			`$EF`,
			map[string]string{
				"AB": "$E",
				"CD": "F",
				"EF": "grr and brr",
			},
		)
	})
}