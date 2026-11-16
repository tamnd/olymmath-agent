package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// grant turns the refused row evan-chen of the fixture into a granted one.
func grant(s string) string {
	s = strings.Replace(s, "  state: refused\n", "  granted: {statements: true, years: 2010-2020, uses: [site]}\n  state: granted\n", 1)
	return strings.Replace(s, "answer: No; links only.", "answer: Yes for the site, with a link to the page of each problem.", 1)
}

// edit applies pre, when it is not nil, to a file of a copy of the fixture and then replaces old with repl in it.
func edit(t *testing.T, root, file, old, repl string, pre func(string) string) {
	t.Helper()
	path := filepath.Join(root, file)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if pre != nil {
		s = pre(s)
	}
	if !strings.Contains(s, old) {
		t.Fatalf("the fixture has no %q", old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(s, old, repl, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPermissionsLoad(t *testing.T) {
	c, fs := Load("testdata/valid")
	if len(fs) > 0 {
		t.Fatalf("want no findings, got:\n%v", fs)
	}
	if p := c.Permissions["evan-chen"]; p == nil || p.State != "refused" || p.Granted != nil {
		t.Fatalf("the row evan-chen is not loaded right: %+v", p)
	}
	if ws := PermissionWarnings(c); len(ws) > 0 {
		t.Fatalf("want no warnings, got:\n%v", ws)
	}

	root := copyTree(t)
	edit(t, root, "registers/permissions.yaml", "priority: P1", "priority: P1", grant)
	c, fs = Load(root)
	if len(fs) > 0 {
		t.Fatalf("want no findings, got:\n%v", fs)
	}
	p := c.Permissions["evan-chen"]
	if p == nil || p.State != "granted" || p.Granted == nil || p.Granted.Years != "2010-2020" {
		t.Fatalf("the row evan-chen is not loaded right: %+v", p)
	}
	ws := PermissionWarnings(c)
	if len(ws) != 1 || ws[0].Level != Warning || !strings.Contains(ws[0].Message, "still in zone Z3") {
		t.Fatalf("want one warning for the grant that is not applied, got:\n%v", ws)
	}
}

func TestPermissionsFindings(t *testing.T) {
	const perm = "registers/permissions.yaml"
	cases := []struct {
		name, file, old, new, want string
		pre                        func(string) string
	}{
		{"unknown competition", perm, "competitions: [imo]", "competitions: [imo, imx]", "the competition \"imx\" is not in registers/competitions.yaml", nil},
		{"unknown source", perm, "sources: [evanchen]", "sources: [evan]", "the source \"evan\" is not in registers/sources.yaml", nil},
		{"same source twice", perm, "sources: [evanchen]", "sources: [evanchen, evanchen]", "is in the row two times", nil},
		{"nothing covered", perm, "  competitions: [imo]\n  sources: [imo-official]\n", "", "covers no competition and no source", nil},
		{"same id", perm, "id: evan-chen", "id: imo-board", "is in the register two times", nil},
		{"bad id", perm, "id: evan-chen", "id: Evan", "is not valid", nil},
		{"no holder", perm, "holder: Evan Chen", "holder: \"\"", "it has no holder", nil},
		{"bad country", perm, "country: US", "country: USA", "not an ISO 3166-1 alpha-2 code", nil},
		{"bad priority", perm, "priority: P0", "priority: P5", "the priority \"P5\"", nil},
		{"bad state", perm, "state: draft", "state: pending", "the state \"pending\"", nil},
		{"zone in register", perm, "priority: P0", "priority: P0\n  zone: Z1", "field zone not found", nil},
		{"empty scope", perm, "scope: {statements: true, solutions: true, years: all, uses: [site, export]}", "scope: {years: all, uses: [site]}", "covers neither statements nor solutions", nil},
		{"bad years", perm, "years: all, uses: [site, export]", "years: 1990-1980, uses: [site, export]", "end before they start", nil},
		{"bad use", perm, "uses: [site, export]", "uses: [site, print]", "the use \"print\"", nil},
		{"no uses", perm, "uses: [site, export]", "uses: []", "has no uses", nil},
		{"free mail", perm, "url: https://web.evanchen.cc/", "url: https://web.evanchen.cc/\n    email: someone@gmail.com", "free-mail domain", nil},
		{"bad date", perm, "drafted: 2026-11-16", "drafted: 2026-13-01", "is not YYYY-MM-DD", nil},
		{"answered before sent", perm, "answered: 2026-11-10", "answered: 2026-11-01", "answered before it was sent", nil},
		{"refused with no answer", perm, "  answer: No; links only.\n", "", "needs the answer", nil},
		{"refused with a grant", perm, "  state: refused\n", "  granted: {statements: true, years: all, uses: [site]}\n  state: refused\n", "a refused request has no granted scope", nil},
		{"granted with no date", perm, ", answered: 2026-11-10}", "}", "needs the date answered", grant},
		{"granted with no scope", perm, "  granted: {statements: true, years: 2010-2020, uses: [site]}\n", "", "a granted request needs the granted scope", grant},
		{"granted with no answer", perm, "  answer: Yes for the site, with a link to the page of each problem.\n", "", "needs the answer", grant},
		{"sent with no letter", perm, "  letter: sha256:0f343b0931126a20f133d67c2b018a3b5b6a9c1e2f3d4c5b6a7980716253443f\n", "", "needs the hash of the letter", nil},
		{"bad letter", perm, "letter: sha256:0f34", "letter: md5:0f34", "is not sha256:", nil},
		{"sent draft", perm, "dates: {drafted: 2026-11-16}", "dates: {drafted: 2026-11-16, sent: 2026-11-17}", "a draft has no sent date", nil},
		{"z1 with no basis", "registers/sources.yaml", "  licence: CC-BY-SA-4.0\n", "", "is in zone Z1 with no licence", nil},
		{"basis not granted", "registers/sources.yaml", "licence: CC-BY-SA-4.0", "licence: permission:imo-board", "which is in state draft, not granted", nil},
		{"basis refused", "registers/sources.yaml", "licence: CC-BY-SA-4.0", "licence: permission:evan-chen", "which is in state refused, not granted", nil},
		{"basis unknown", "registers/sources.yaml", "licence: CC-BY-SA-4.0", "licence: permission:nobody", "which is not in registers/permissions.yaml", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyTree(t)
			if tc.pre != nil {
				// The edit of the register comes first, so a case on another file sees the granted row.
				edit(t, root, perm, "priority: P1", "priority: P1", tc.pre)
			}
			edit(t, root, tc.file, tc.old, tc.new, nil)
			_, fs := Load(root)
			if !strings.Contains(fs.Error(), tc.want) {
				t.Fatalf("want a finding with %q, got:\n%v", tc.want, fs)
			}
		})
	}
}

func TestPermissionsBasisNotCovering(t *testing.T) {
	root := copyTree(t)
	edit(t, root, "registers/permissions.yaml", "priority: P1", "priority: P1", grant)
	edit(t, root, "registers/sources.yaml", "licence: CC-BY-SA-4.0", "licence: permission:evan-chen", nil)
	_, fs := Load(root)
	if !strings.Contains(fs.Error(), "which does not cover it") {
		t.Fatalf("want a finding for a grant that does not cover the source, got:\n%v", fs)
	}
}

func TestPermissionsApplied(t *testing.T) {
	root := copyTree(t)
	edit(t, root, "registers/permissions.yaml", "priority: P1", "priority: P1", grant)
	edit(t, root, "registers/sources.yaml", "  access: tex\n  zone: {statement: Z3, solution: Z3}", "  access: tex\n  licence: permission:evan-chen\n  zone: {statement: Z1, solution: Z3}", nil)
	c, fs := Load(root)
	if len(fs) > 0 {
		t.Fatalf("want no findings, got:\n%v", fs)
	}
	if ws := PermissionWarnings(c); len(ws) > 0 {
		t.Fatalf("want no warnings once the grant is applied, got:\n%v", ws)
	}
}
