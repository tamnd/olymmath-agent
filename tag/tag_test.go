package tag

import (
	"errors"
	"strings"
	"testing"
)

func TestMintMillion(t *testing.T) {
	n := 1_000_000
	if testing.Short() {
		n = 10_000
	}
	seen := make(map[string]bool, n)
	used := func(s string) bool { return seen[s] }
	for range n {
		got, err := Mint(nil, used)
		if err != nil {
			t.Fatal(err)
		}
		if !IsTag(got) || len(got) != Len || got[0] == '0' || Excluded(got) || strings.ContainsAny(got, "ILOU") {
			t.Fatalf("Mint gave %q", got)
		}
		seen[got] = true
	}
	if len(seen) != n {
		t.Fatalf("got %d different tags, want %d", len(seen), n)
	}
}

func TestMintSeed(t *testing.T) {
	for range 1000 {
		got, err := MintSeed(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !IsSeed(got) {
			t.Fatalf("MintSeed gave %q", got)
		}
	}
}

func TestMintExhausted(t *testing.T) {
	_, err := Mint(nil, func(string) bool { return true })
	if !errors.Is(err, ErrExhausted) {
		t.Fatalf("got %v, want ErrExhausted", err)
	}
}

func TestEncodeUsesEachCharacter(t *testing.T) {
	for v := range uint64(32) {
		if got, want := encode(v), "00000000"+string(Alphabet[v]); got != want {
			t.Fatalf("encode(%d) = %q, want %q", v, got, want)
		}
	}
	if got := encode(1<<45 - 1); got != "ZZZZZZZZZ" {
		t.Fatalf("encode(max) = %q", got)
	}
}

func TestCheck(t *testing.T) {
	if got := Check("7K3QX2M9R"); got != '2' {
		t.Fatalf("Check(7K3QX2M9R) = %q, want '2'", got)
	}
	if got := Check("7K3QX2M9U"); got != 0 {
		t.Fatalf("Check of a payload with U = %q, want 0", got)
	}
	// Each change of one character and each swap of two neighbours that differ must fail the check.
	good := "7K3QX2M9R2"
	for i := range len(good) {
		for _, c := range []byte(Alphabet) {
			if c == good[i] {
				continue
			}
			b := []byte(good)
			b[i] = c
			if IsTag(string(b)) {
				t.Fatalf("IsTag(%s) = true after a change of character %d", b, i)
			}
		}
		if i+1 < len(good) && good[i] != good[i+1] {
			b := []byte(good)
			b[i], b[i+1] = b[i+1], b[i]
			if IsTag(string(b)) {
				t.Fatalf("IsTag(%s) = true after a swap at %d", b, i)
			}
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want, err string }{
		{"7K3QX2M9R2", "7K3QX2M9R2", ""},
		{"7k3qx-2m9r2", "7K3QX2M9R2", ""},
		{" 7K3QX 2M9R2 ", "7K3QX2M9R2", ""},
		{"7k3qx2m9r2", "7K3QX2M9R2", ""},
		{"7K3XQ2M9R2", "", "check character"},
		{"7KZQX2M9R2", "", "check character"},
		{"7K3QX2M9", "", "not a valid tag"},
		{"7K3QX2M9RU", "", "not a valid tag"},
	}
	for _, c := range cases {
		got, err := Normalize(c.in)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("Normalize(%q) = %q, %v; want an error with %q", c.in, got, err, c.err)
			}
			continue
		}
		if got != c.want || err != nil {
			t.Errorf("Normalize(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := Normalize("7K3XQ2M9R2"); !errors.Is(err, ErrCheck) {
		t.Errorf("a wrong check character gave %v, want ErrCheck", err)
	}
}

func TestDisplay(t *testing.T) {
	if got := Display("7K3QX2M9R2"); got != "7K3QX-2M9R2" {
		t.Errorf("Display = %q", got)
	}
}

func TestExcluded(t *testing.T) {
	for _, s := range []string{"1234567890", "12E3456789", "20D60XXXXX", "123456789X", "A123456789"} {
		if !Excluded(s) {
			t.Errorf("Excluded(%q) = false", s)
		}
	}
	for _, s := range []string{"7K3QX2M9R2", "AB12345678"} {
		if Excluded(s) {
			t.Errorf("Excluded(%q) = true", s)
		}
	}
}

func TestShard(t *testing.T) {
	if got := LoosePath("7K3QX2M9R2"); got != "problems/_loose/7K3/7K3QX2M9R2.md" {
		t.Errorf("LoosePath = %q", got)
	}
	if got := Shard("7K"); got != "7K" {
		t.Errorf("Shard of a short string = %q", got)
	}
}
