package sl_test

import (
	"testing"

	"github.com/hookkster/pkg/logger/sl"
)

func TestMask(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"ignat@mail.ru": "i****@mail.ru",
		"a@mail.ru":     "*@mail.ru",
		"@mail.ru":      "@mail.ru",
		"89344443321":   "89*******21",
		"1234":          "****",
		"1":             "*",
	}

	for in, want := range cases {
		if got := sl.Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}
