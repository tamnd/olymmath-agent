// Package katex renders TeX math to HTML and MathML with KaTeX, with no browser and no Node.js.
//
// The package embeds the minified script of one KaTeX release and runs it in goja, a JavaScript engine in pure Go. Render returns the same markup that katex.renderToString returns in a browser. The markup has two parts: HTML that looks the same in each browser, and MathML for screen readers. Thus a page shows its math with JavaScript off.
//
// The HTML part needs the style sheet and the fonts of the same release. Assets returns them. Serve them next to the pages, and link the style sheet from each page.
//
// Run go generate to change the release. The generator checks the tarball from npm against its published integrity.
package katex

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

//go:generate go run gen.go

// Version is the KaTeX release that the package embeds.
const Version = "0.19.1"

//go:embed katex.min.js
var script string

//go:embed assets
var assets embed.FS

// Assets returns the files that the HTML of Render needs: katex.min.css, the fonts in fonts/ and the licence of KaTeX.
func Assets() fs.FS {
	sub, _ := fs.Sub(assets, "assets")
	return sub
}

// Timeout is the longest time that one call of Render can take.
const Timeout = 5 * time.Second

// prelude makes a global object for the script and a render function with fixed options. Each call gets a new copy of the macros, so that a \gdef in one formula does not change the next formula.
const prelude = `var window = this, module = {exports: {}}, exports = module.exports, omTokenRegex;`

const glue = `
var omkatex = module.exports.renderToString ? module.exports : katex;
function omrender(tex, display, macros) {
	return omkatex.renderToString(tex, {
		displayMode: display,
		output: "htmlAndMathml",
		throwOnError: true,
		strict: "ignore",
		trust: false,
		maxExpand: 1000,
		macros: Object.assign({}, macros)
	});
}`

// Error is a formula that KaTeX cannot parse.
type Error struct {
	TeX     string // the formula
	Message string // the message of KaTeX
}

func (e *Error) Error() string { return e.Message }

// lexerNew is the part of the script that makes the regular expression of the lexer. KaTeX makes a new lexer for each formula and each macro, and goja compiles the expression again each time, which was about half the time of a render. The patch compiles it once, and each lexer gets a copy that shares the compiled form and has its own lastIndex.
const (
	lexerNew  = `this.tokenRegex=new RegExp(Xn,"g")`
	lexerOnce = `this.tokenRegex=new RegExp(omTokenRegex||(omTokenRegex=new RegExp(Xn,"g")))`
)

// Renderer renders formulas with one set of macros. It keeps the result of each formula, so a formula that a site uses many times costs one render. Its methods are safe for concurrent use: it runs up to runtime.NumCPU KaTeX engines at a time, and each engine renders one formula at a time.
type Renderer struct {
	macros map[string]any
	name   string        // the name of the cache file
	idle   chan *engine  // the engines that render nothing now
	slots  chan struct{} // one token for each engine that may still start

	mu    sync.Mutex
	cache map[key]result
}

// engine is one JavaScript runtime with KaTeX loaded.
type engine struct {
	vm     *goja.Runtime
	render goja.Callable
	macros goja.Value
}

type key struct {
	tex     string
	display bool
}

type result struct {
	html    string
	err     error
	timeout bool // the render took too long. This depends on the load of the machine, so the cache file does not keep it
	used    bool // the formula was asked for since New, so the cache file keeps it
}

// New returns a Renderer with the macros. A key of macros is a control sequence such as \R, and its value is the expansion, with #1 for the first argument.
func New(macros map[string]string) (*Renderer, error) {
	m := map[string]any{}
	for k, v := range macros {
		m[k] = v
	}
	n := max(runtime.NumCPU(), 1)
	r := &Renderer{macros: m, name: cacheName(macros), idle: make(chan *engine, n), slots: make(chan struct{}, n-1), cache: map[key]result{}}
	for range n - 1 {
		r.slots <- struct{}{}
	}
	// The first engine starts now, so that a broken script is an error of New.
	e, err := r.newEngine()
	if err != nil {
		return nil, err
	}
	r.idle <- e
	return r, nil
}

func (r *Renderer) newEngine() (*engine, error) {
	vm := goja.New()
	src := strings.Replace(script, lexerNew, lexerOnce, 1)
	if _, err := vm.RunScript("katex.min.js", prelude+"\n"+src+"\n"+glue); err != nil {
		return nil, fmt.Errorf("katex: load KaTeX %s: %w", Version, err)
	}
	render, ok := goja.AssertFunction(vm.Get("omrender"))
	if !ok {
		return nil, errors.New("katex: the script has no render function")
	}
	return &engine{vm: vm, render: render, macros: vm.ToValue(r.macros)}, nil
}

// get returns an idle engine. It starts a new engine when none is idle and fewer than the limit run.
func (r *Renderer) get() (*engine, error) {
	select {
	case e := <-r.idle:
		return e, nil
	default:
	}
	select {
	case e := <-r.idle:
		return e, nil
	case <-r.slots:
		e, err := r.newEngine()
		if err != nil {
			r.slots <- struct{}{}
			return nil, err
		}
		return e, nil
	}
}

// Render returns the HTML and MathML of tex. display selects display math, else the formula is inline. The error is an *Error when KaTeX cannot parse tex.
func (r *Renderer) Render(tex string, display bool) (string, error) {
	k := key{tex, display}
	r.mu.Lock()
	res, ok := r.cache[k]
	if ok && !res.used {
		res.used = true
		r.cache[k] = res
	}
	r.mu.Unlock()
	if ok {
		return res.html, res.err
	}
	e, err := r.get()
	if err != nil {
		return "", err
	}
	res = e.run(tex, display)
	res.used = true
	r.idle <- e
	r.mu.Lock()
	r.cache[k] = res
	r.mu.Unlock()
	return res.html, res.err
}

func (e *engine) run(tex string, display bool) result {
	t := time.AfterFunc(Timeout, func() { e.vm.Interrupt("timeout") })
	v, err := e.render(goja.Undefined(), e.vm.ToValue(tex), e.vm.ToValue(display), e.macros)
	t.Stop()
	e.vm.ClearInterrupt()
	var res result
	switch {
	case err == nil:
		res.html = v.String()
	case isInterrupt(err):
		res.err = &Error{TeX: tex, Message: fmt.Sprintf("KaTeX took more than %s", Timeout)}
		res.timeout = true
	default:
		res.err = &Error{TeX: tex, Message: message(err)}
	}
	return res
}

func isInterrupt(err error) bool {
	var ie *goja.InterruptedError
	return errors.As(err, &ie)
}

// message returns the message of a JavaScript exception, without the stack.
func message(err error) string {
	var ex *goja.Exception
	if errors.As(err, &ex) {
		if o, ok := ex.Value().(*goja.Object); ok {
			if m := o.Get("message"); m != nil && !goja.IsUndefined(m) {
				return m.String()
			}
		}
		return strings.TrimSpace(ex.Value().String())
	}
	return err.Error()
}
