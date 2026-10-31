// The small scripts of the site: the theme button, the keys of the problem pages, the print button and the search. Each page works without them, except the search.
(function () {
  "use strict";
  var root = document.documentElement;

  function setTheme(t) {
    root.setAttribute("data-theme", t);
    try { localStorage.setItem("theme", t); } catch (e) {}
  }
  function toggleTheme() {
    setTheme(root.getAttribute("data-theme") === "dark" ? "light" : "dark");
  }
  var btn = document.querySelector(".theme");
  if (btn) btn.addEventListener("click", toggleTheme);

  var printBtn = document.querySelector("[data-print]");
  if (printBtn) printBtn.addEventListener("click", function () { window.print(); });

  var problem = document.querySelector(".problem");
  document.addEventListener("keydown", function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    var tag = (e.target.tagName || "").toLowerCase();
    if (tag === "input" || tag === "textarea" || e.target.isContentEditable) return;
    if (e.key === "t") { toggleTheme(); return; }
    if (e.key === "/") {
      e.preventDefault();
      location.href = document.body.getAttribute("data-base") + "search/";
      return;
    }
    if (!problem) return;
    var to = e.key === "j" ? problem.getAttribute("data-next") : e.key === "k" ? problem.getAttribute("data-prev") : null;
    if (to) location.href = to;
  });

  // Search: the index is one JSON file. Each word of the query must be in the title, the locator or the text.
  var results = document.getElementById("results");
  if (!results) return;
  var input = document.getElementById("q");
  var status = document.getElementById("search-status");
  var index = null;

  function norm(s) {
    return s.toLowerCase().normalize("NFKD").replace(/[\u0300-\u036f]/g, "");
  }
  function escapeHTML(s) {
    return s.replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }
  function snippet(text, words) {
    var n = norm(text), at = 0;
    for (var i = 0; i < words.length; i++) {
      var k = n.indexOf(words[i]);
      if (k >= 0) { at = Math.max(0, k - 60); break; }
    }
    var s = (at > 0 ? "…" : "") + text.slice(at, at + 220) + (at + 220 < text.length ? "…" : "");
    var out = escapeHTML(s);
    words.forEach(function (w) {
      if (w.length < 2) return;
      out = out.replace(new RegExp("(" + w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + ")", "gi"), "<mark>$1</mark>");
    });
    return out;
  }
  // formulaTokens follows FormulaTokens of the site package: a command gives its name, and a letter, a digit or an operator gives itself.
  var skip = " left right big Big bigg Bigg bigl bigr Bigl Bigr displaystyle textstyle mathrm mathbf mathit text operatorname quad qquad ";
  var same = { leq: "le", geq: "ge", neq: "ne", dfrac: "frac", tfrac: "frac", lvert: "|", rvert: "|", vert: "|", mid: "|", ldots: "dots", cdots: "dots", leqslant: "le", geqslant: "ge" };
  function formulaTokens(tex) {
    var out = [], i = 0, cs = Array.from(tex);
    while (i < cs.length) {
      var c = cs[i];
      if (c === "\\") {
        var j = i + 1;
        while (j < cs.length && /[a-zA-Z]/.test(cs[j])) j++;
        var name = cs.slice(i + 1, j).join("");
        if (!name) {
          if (j < cs.length && "{}|".indexOf(cs[j]) >= 0) out.push(cs[j]);
          j++;
        } else if (skip.indexOf(" " + name + " ") < 0) {
          out.push(same[name] || name);
        }
        i = j;
      } else if (/[a-zA-Z0-9]/.test(c) || "+-=<>()[]|,.!/*:;".indexOf(c) >= 0) {
        out.push(c); i++;
      } else {
        if (c.charCodeAt(0) >= 128 && !/\s/.test(c)) out.push(c);
        i++;
      }
    }
    return out.join(" ");
  }
  // isFormula tells if a query is a formula: it has a TeX command, a script, a brace, a dollar or an equals sign.
  function isFormula(q) { return /[\\^_{}$=]/.test(q); }
  function run() {
    var q = input.value.trim();
    var url = new URL(location.href);
    if (q) url.searchParams.set("q", q); else url.searchParams.delete("q");
    history.replaceState(null, "", url);
    results.innerHTML = "";
    if (!q) { status.textContent = "Type words to search the statements."; return; }
    var words = norm(q).split(/\s+/).filter(Boolean), hits;
    if (isFormula(q)) {
      var f = " " + formulaTokens(q.replace(/\$/g, "")) + " ";
      words = [];
      hits = index.filter(function (e) { return e.f && (" " + e.f + " ").indexOf(f) >= 0; });
    } else {
      hits = index.filter(function (e) {
        var hay = norm(e.t + " " + e.l + " " + e.x);
        return words.every(function (w) { return hay.indexOf(w) >= 0; });
      });
    }
    hits.sort(function (a, b) { return b.y - a.y || (a.l < b.l ? -1 : 1); });
    status.textContent = hits.length === 1 ? "1 problem" : hits.length + " problems";
    var frag = document.createDocumentFragment();
    hits.slice(0, 100).forEach(function (e) {
      var li = document.createElement("li");
      li.innerHTML = '<a class="card" href="' + escapeHTML(e.u) + '"><div class="r-title">' + escapeHTML(e.t) + '</div><div class="r-text">' + snippet(e.x, words) + "</div></a>";
      frag.appendChild(li);
    });
    results.appendChild(frag);
  }
  var q0 = new URL(location.href).searchParams.get("q");
  if (q0) input.value = q0;
  status.textContent = "Loading the index…";
  fetch(results.getAttribute("data-index")).then(function (r) { return r.json(); }).then(function (data) {
    index = data;
    var exact = index.filter(function (e) { return e.l.toLowerCase() === input.value.trim().toLowerCase(); });
    if (exact.length === 1) { location.replace(exact[0].u); return; }
    run();
    input.addEventListener("input", run);
    // A locator or a tag goes straight to its page.
    document.querySelector("[data-search]").addEventListener("submit", function (e) {
      e.preventDefault();
      var q = input.value.trim().toLowerCase().replace(/^\/+|\/+$/g, "");
      var hit = index.filter(function (x) { return x.l.toLowerCase() === q; });
      if (hit.length === 1) { location.href = hit[0].u; return; }
      if (/^[0-9a-z]{10}$/.test(q) && /\d/.test(q)) { location.href = document.body.getAttribute("data-base") + "p/" + q.toUpperCase(); return; }
      run();
    });
  }).catch(function () { status.textContent = "The search index did not load. Try again later."; });
  // The filter of the competition list hides each row that does not contain the text, and each group that has no row left.
  var filter = document.querySelector("[data-filter]");
  if (filter) {
    filter.addEventListener("input", function () {
      var q = filter.value.trim().toLowerCase();
      var shown = 0;
      document.querySelectorAll("[data-row]").forEach(function (row) {
        var hit = row.getAttribute("data-text").toLowerCase().indexOf(q) >= 0;
        row.hidden = !hit;
        if (hit) shown++;
      });
      document.querySelectorAll("[data-group]").forEach(function (g) {
        g.hidden = !g.querySelector("[data-row]:not([hidden])");
      });
      document.querySelector("[data-empty]").hidden = shown > 0;
    });
  }
})();
