package official

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// A PDF that Ghostscript made from the output of dvips can set its Russian text in Type 3 fonts in the T2A encoding of TeX. The ToUnicode map of such a font sends some letters, as Г, Р and ф, to control codes, and gives no map for the dashes and the quotes, which sit at control codes in T2A. pdftohtml then drops these letters and signs. pdfLost reads the codes of the Type 3 fonts from the content of the pages and keeps the letters and signs that the text layer loses, with the letters and digits around them, so that pdfRepair can put them back.

var (
	rxObj       = regexp.MustCompile(`(?s)(\d+) 0 obj(.*?)endobj`)
	rxStream    = regexp.MustCompile(`(?s)stream\r?\n(.*)endstream`)
	rxRef       = regexp.MustCompile(`/(\w+)\s+(\d+)\s+0\s+R`)
	rxPageObj   = regexp.MustCompile(`/Type\s*/Page[^s]`)
	rxFontRef   = regexp.MustCompile(`/Font\s*(\d+)\s+0\s+R`)
	rxFontDict  = regexp.MustCompile(`(?s)/Font\s*<<(.*?)>>`)
	rxContents  = regexp.MustCompile(`/Contents\s*(?:(\d+)\s+0\s+R|\[([^\]]*)\])`)
	rxToUnicode = regexp.MustCompile(`/ToUnicode\s*(\d+)\s+0\s+R`)
	rxBfRange   = regexp.MustCompile(`<([0-9a-fA-F]{2})>\s*<([0-9a-fA-F]{2})>\s*<([0-9a-fA-F]{4})>`)
	rxBfChar    = regexp.MustCompile(`(?m)^\s*<([0-9a-fA-F]{2})>\s*<([0-9a-fA-F]{4})>\s*$`)
	rxNum       = regexp.MustCompile(`\d+`)
)

// pdfT2A gives the letters and signs of the T2A encoding that are not the same as in Windows-1251.
var pdfT2A = map[byte]rune{0x10: '“', 0x11: '”', 0x12: '„', 0x13: '«', 0x14: '»', 0x15: '–', 0x16: '—', 0x9c: 'Ё', 0xbc: 'ё'}

// pdfLostKey is a lost letter or sign: the letters and digits before and after it.
type pdfLostKey struct{ before, after string }

// pdfLostSign is what a pdfLostKey gives: the letter or sign, and whether it is a part of the word before or after it.
type pdfLostSign struct {
	text                string
	letter, left, right bool
	n                   int
}

// pdfLostSet is the lost letters and signs of a PDF.
type pdfLostSet struct {
	signs map[pdfLostKey]*pdfLostSign
	// unknown counts the lost codes that are not in T2A, so that their letters are not known.
	unknown int
}

const pdfKeyLen = 8

// pdfKeyRune tells if a letter or digit is a part of a key: a Latin or Russian letter or a digit.
func pdfKeyRune(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= 'А' && r <= 'я' || r == 'ё' || r == 'Ё'
}

// pdfRawRune is one letter of the content of a page: the rune, and whether the text layer loses it.
type pdfRawRune struct {
	r    rune
	lost bool
	// gap is true before the first letter of a piece of text that the content of the page shows on its own.
	gap bool
}

// pdfLost reads the lost letters and signs of a PDF. It gives nil when the PDF has no Type 3 font that loses one.
func pdfLost(pdf []byte) *pdfLostSet {
	objs := map[string][]byte{}
	var order []string
	for _, m := range rxObj.FindAllSubmatch(pdf, -1) {
		id := string(m[1])
		if _, ok := objs[id]; !ok {
			order = append(order, id)
		}
		objs[id] = m[2]
	}
	// lostCodes gives, for each Type 3 font, the codes that its text layer loses.
	lostCodes := map[string]map[byte]bool{}
	for id, body := range objs {
		head := body
		if i := bytes.Index(body, []byte("stream")); i >= 0 {
			head = body[:i]
		}
		if !bytes.Contains(head, []byte("/CharProcs")) {
			continue
		}
		cmap := map[byte]rune{}
		if m := rxToUnicode.FindSubmatch(head); m != nil {
			s := pdfStream(objs[string(m[1])])
			for _, r := range rxBfRange.FindAllSubmatch(s, -1) {
				a, _ := strconv.ParseUint(string(r[1]), 16, 8)
				b, _ := strconv.ParseUint(string(r[2]), 16, 8)
				c, _ := strconv.ParseUint(string(r[3]), 16, 32)
				for x := a; x <= b; x++ {
					cmap[byte(x)] = rune(c + x - a) // #nosec G115 -- x is a code of 8 bits and c a code of 16 bits.
				}
			}
			for _, r := range rxBfChar.FindAllSubmatch(s, -1) {
				a, _ := strconv.ParseUint(string(r[1]), 16, 8)
				c, _ := strconv.ParseUint(string(r[2]), 16, 32)
				cmap[byte(a)] = rune(c) // #nosec G115 -- the code is 8 bits and the rune 16 bits.
			}
		}
		lost := map[byte]bool{}
		for b := 0; b < 256; b++ {
			if u, ok := cmap[byte(b)]; ok && u < 0x20 || !ok && b < 0x20 {
				lost[byte(b)] = true
			}
		}
		lostCodes[id] = lost
	}
	if len(lostCodes) == 0 {
		return nil
	}
	set := &pdfLostSet{signs: map[pdfLostKey]*pdfLostSign{}}
	for _, id := range order {
		head := objs[id]
		if !rxPageObj.Match(head) {
			continue
		}
		fonts := map[string]string{}
		dict := head
		if m := rxFontRef.FindSubmatch(head); m != nil {
			dict = objs[string(m[1])]
		} else if m := rxFontDict.FindSubmatch(head); m != nil {
			dict = m[1]
		}
		for _, r := range rxRef.FindAllSubmatch(dict, -1) {
			fonts[string(r[1])] = string(r[2])
		}
		m := rxContents.FindSubmatch(head)
		if m == nil {
			continue
		}
		var ids []string
		if len(m[1]) > 0 {
			ids = []string{string(m[1])}
		} else {
			for i, n := range rxNum.FindAll(m[2], -1) {
				// The list holds a reference as "N 0 R", so every third number is an object.
				if i%3 == 0 {
					ids = append(ids, string(n))
				}
			}
		}
		var content []byte
		for _, c := range ids {
			content = append(content, pdfStream(objs[c])...)
			content = append(content, '\n')
		}
		set.add(pdfContentRunes(content, func(name string) map[byte]bool {
			return lostCodes[fonts[name]]
		}))
	}
	if len(set.signs) == 0 && set.unknown == 0 {
		return nil
	}
	return set
}

// add keeps the lost letters and signs of the letters of a page. Lost letters next to each other, as the two letters of "фф", make one group.
func (set *pdfLostSet) add(rs []pdfRawRune) {
	for i := 0; i < len(rs); i++ {
		if !rs[i].lost {
			continue
		}
		j := i
		for j < len(rs) && rs[j].lost {
			j++
		}
		var group []rune
		for _, x := range rs[i:j] {
			group = append(group, x.r)
		}
		first, last := rs[i], rs[j-1]
		start := i
		i = j - 1
		if slices.Contains(group, unicode.ReplacementChar) {
			set.unknown++
			continue
		}
		var before, after []rune
		for k := start - 1; k >= 0 && len(before) < pdfKeyLen; k-- {
			if !rs[k].lost && pdfKeyRune(rs[k].r) {
				before = append([]rune{rs[k].r}, before...)
			}
		}
		for k := j; k < len(rs) && len(after) < pdfKeyLen; k++ {
			if !rs[k].lost && pdfKeyRune(rs[k].r) {
				after = append(after, rs[k].r)
			}
		}
		key := pdfLostKey{string(before), string(after)}
		s := set.signs[key]
		if s == nil {
			s = &pdfLostSign{text: string(group), letter: unicode.IsLetter(first.r)}
			set.signs[key] = s
		} else if s.text != string(group) {
			// Two different signs at the same place cannot be told apart.
			set.unknown++
			continue
		}
		s.n++
		if s.letter {
			s.left = s.left || start > 0 && !first.gap && unicode.IsLetter(rs[start-1].r)
			// A capital letter starts a word. A letter that is a piece of its own goes into the word after it.
			s.right = s.right || unicode.IsUpper(first.r) || !s.left || j < len(rs) && !rs[j].gap && unicode.IsLetter(rs[j].r)
		} else {
			s.right = strings.ContainsRune("«„“", last.r)
			s.left = strings.ContainsRune("»”", first.r)
		}
	}
}

// pdfStream gives the stream of an object, inflated when it is compressed.
func pdfStream(body []byte) []byte {
	m := rxStream.FindSubmatch(body)
	if m == nil {
		return nil
	}
	raw := m[1]
	if !bytes.Contains(body[:bytes.Index(body, []byte("stream"))], []byte("FlateDecode")) {
		return raw
	}
	r, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	out, _ := io.ReadAll(r)
	return out
}

// pdfContentRunes reads the letters that a content stream shows, in their order. lost gives the codes that the text layer loses for a font, or nil for a font that is not a Type 3 font. The text of a font that is not a Type 3 font is read as Latin-1, which is the case for the letters and digits of the fonts of TeX.
func pdfContentRunes(c []byte, lost func(string) map[byte]bool) []pdfRawRune {
	var out []pdfRawRune
	var font, name string
	var strs [][]byte
	dec := charmap.Windows1251.NewDecoder()
	for i := 0; i < len(c); {
		ch := c[i]
		switch {
		case ch == '(':
			s, n := pdfLiteral(c[i:])
			strs = append(strs, s)
			i += n
		case ch == '<' && i+1 < len(c) && c[i+1] != '<':
			j := bytes.IndexByte(c[i:], '>')
			if j < 0 {
				return out
			}
			s := make([]byte, 0, j/2)
			hex := bytes.Map(func(r rune) rune {
				if unicode.Is(unicode.ASCII_Hex_Digit, r) {
					return r
				}
				return -1
			}, c[i+1:i+j])
			if len(hex)%2 == 1 {
				hex = append(hex, '0')
			}
			for k := 0; k+1 < len(hex); k += 2 {
				v, _ := strconv.ParseUint(string(hex[k:k+2]), 16, 8)
				s = append(s, byte(v))
			}
			strs = append(strs, s)
			i += j + 1
		case ch == '/':
			j := i + 1
			for j < len(c) && !pdfDelim(c[j]) {
				j++
			}
			name = string(c[i+1 : j])
			i = j
		case ch == '%':
			for i < len(c) && c[i] != '\n' && c[i] != '\r' {
				i++
			}
		case ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '\'' || ch == '"':
			j := i + 1
			for j < len(c) && !pdfDelim(c[j]) && c[j] != '\'' && c[j] != '"' {
				j++
			}
			op := string(c[i:j])
			i = j
			switch op {
			case "Tf":
				font = name
			case "Tj", "TJ", "'", "\"":
				codes := lost(font)
				for k, s := range strs {
					for x, b := range s {
						rr := pdfRawRune{gap: x == 0 && (k == 0 || op == "TJ")}
						if codes == nil {
							rr.r = rune(b)
						} else {
							if r, ok := pdfT2A[b]; ok {
								rr.r = r
							} else if b >= 0x20 {
								d, _ := dec.Bytes([]byte{b})
								rr.r = []rune(string(d))[0]
							} else {
								rr.r = unicode.ReplacementChar
							}
							rr.lost = codes[b]
						}
						out = append(out, rr)
					}
				}
			case "BI":
				// An image in the content: skip its data.
				k := bytes.Index(c[i:], []byte("EI"))
				if k < 0 {
					return out
				}
				i += k + 2
			}
			strs = nil
		default:
			i++
		}
	}
	return out
}

func pdfDelim(b byte) bool {
	return bytes.IndexByte([]byte(" \t\r\n\f\x00()<>[]{}/%"), b) >= 0
}

// pdfLiteral reads a literal string that starts at s[0], which is "(". It gives the bytes of the string and the length of the literal.
func pdfLiteral(s []byte) ([]byte, int) {
	var out []byte
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '(':
			if depth > 0 {
				out = append(out, ch)
			}
			depth++
		case ')':
			depth--
			if depth == 0 {
				return out, i + 1
			}
			out = append(out, ch)
		case '\\':
			i++
			if i >= len(s) {
				return out, i
			}
			e := s[i]
			switch e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '\r', '\n':
				if e == '\r' && i+1 < len(s) && s[i+1] == '\n' {
					i++
				}
			default:
				if e >= '0' && e <= '7' {
					v := 0
					k := 0
					for ; k < 3 && i+k < len(s) && s[i+k] >= '0' && s[i+k] <= '7'; k++ {
						v = v*8 + int(s[i+k]-'0')
					}
					i += k - 1
					out = append(out, byte(v))
				} else {
					out = append(out, e)
				}
			}
		default:
			out = append(out, ch)
		}
	}
	return out, len(s)
}

// pdfPos is the place of one letter of the text layer: the line, the piece and the byte in its text.
type pdfPos struct{ line, frag, at int }

// pdfRepair puts the lost letters and signs back into the lines of the text layer. A letter goes into the word before or after it; a dash goes after the piece before it as a piece of its own. It gives the lost letters and signs that it did not find a place for.
func pdfRepair(lines []pdfLine, set *pdfLostSet) []pdfLostKey {
	if set == nil {
		return nil
	}
	var keys []rune
	var pos []pdfPos
	for li, l := range lines {
		for fi, f := range l.frags {
			for at, r := range f.text {
				if pdfKeyRune(r) {
					keys = append(keys, r)
					pos = append(pos, pdfPos{li, fi, at})
				}
			}
		}
	}
	type insert struct {
		p      pdfPos
		text   string
		newBit bool
	}
	key := func(i int) pdfLostKey {
		lo := max(0, i+1-pdfKeyLen)
		hi := min(len(keys), i+1+pdfKeyLen)
		return pdfLostKey{string(keys[lo : i+1]), string(keys[i+1 : hi])}
	}
	// A place is good only when its context is unique: a context found more often than the sign is lost gives no place.
	found := map[pdfLostKey]int{}
	for i := range keys {
		if k := key(i); set.signs[k] != nil {
			found[k]++
		}
	}
	var ins []insert
	used := map[pdfLostKey]int{}
	for i := range keys {
		k := key(i)
		s := set.signs[k]
		if s == nil || used[k] >= s.n || found[k] > s.n {
			continue
		}
		p := pos[i]
		var q pdfPos
		if i+1 < len(pos) {
			q = pos[i+1]
		} else {
			q = pdfPos{p.line, p.frag, len(lines[p.line].frags[p.frag].text)}
		}
		used[k]++
		switch {
		case p.line == q.line && p.frag == q.frag && s.left && !s.right:
			_, w := utf8DecodeAt(lines[p.line].frags[p.frag].text, p.at)
			ins = append(ins, insert{p: pdfPos{p.line, p.frag, p.at + w}, text: s.text})
		case p.line == q.line && p.frag == q.frag:
			ins = append(ins, insert{p: q, text: s.text, newBit: !s.letter && !s.right})
		case s.right:
			ins = append(ins, insert{p: q, text: s.text})
		case s.left:
			_, w := utf8DecodeAt(lines[p.line].frags[p.frag].text, p.at)
			ins = append(ins, insert{p: pdfPos{p.line, p.frag, p.at + w}, text: s.text})
		case q.line == p.line:
			// A dash goes just before the word after it, as a piece of its own.
			ins = append(ins, insert{p: pdfPos{q.line, q.frag, -2}, text: s.text, newBit: true})
		default:
			// A dash at the end of a line goes after the last piece of the line.
			ins = append(ins, insert{p: pdfPos{p.line, len(lines[p.line].frags) - 1, -1}, text: s.text, newBit: true})
		}
	}
	// Put the letters in from the end, so that the places before them stay good.
	for j := len(ins) - 1; j >= 0; j-- {
		x := ins[j]
		l := &lines[x.p.line]
		f := &l.frags[x.p.frag]
		switch {
		case x.p.at < 0:
			n := pdfFrag{x0: f.x1 + 3, x1: f.x1 + 3, top: f.top, h: f.h, mid: l.mid, text: x.text, kind: 't'}
			at := x.p.frag + 1
			if x.p.at == -2 {
				n.x0, n.x1 = f.x0-3, f.x0-3
				at = x.p.frag
			}
			l.frags = slices.Insert(l.frags, at, n)
		case x.newBit:
			f.text = f.text[:x.p.at] + " " + x.text + " " + f.text[x.p.at:]
		default:
			f.text = f.text[:x.p.at] + x.text + f.text[x.p.at:]
		}
	}
	var missed []pdfLostKey
	for k, s := range set.signs {
		for range s.n - used[k] {
			missed = append(missed, k)
		}
	}
	return missed
}

// pdfKeys gives the letters and digits of a text that make keys.
func pdfKeys(s string) string {
	return strings.Map(func(r rune) rune {
		if pdfKeyRune(r) {
			return r
		}
		return -1
	}, s)
}

func utf8DecodeAt(s string, at int) (rune, int) {
	return utf8.DecodeRuneInString(s[at:])
}
