#!/bin/bash
# check.sh LIB TARGETS OUT builds the modules in TARGETS and runs the checks of spec 08.4 and 08.5 on their declarations. Run it in the directory of the Lake package. TARGETS has lines of module, declaration and kind, separated by tabs. LIB names the output files in the directory OUT. om formal record reads them.
set -u
lib=$1
targets=$2
out=$3
mkdir -p "$out"

# Build the modules, not the library, so that a source builds only the files that have statements. Lake goes on after a failure and reports each failed module.
cut -f1 "$targets" | sort -u | sed 's/^/+/' | xargs lake build > "$out/$lib.build.log" 2>&1
echo "exit $?" >> "$out/$lib.build.log"

# The modules that compiled are the modules that have an .olean file. A Mathlib Archive module is in the Mathlib dependency.
: > "$out/$lib.built"
cut -f1 "$targets" | sort -u | while read -r m; do
  f="lib/lean/$(echo "$m" | tr . / | sed 's/[«»]//g').olean"
  if [ -f ".lake/build/$f" ] || [ -f ".lake/packages/mathlib/.lake/build/$f" ]; then echo "$m" >> "$out/$lib.built"; fi
done
echo "built $(wc -l < "$out/$lib.built") of $(cut -f1 "$targets" | sort -u | wc -l) modules"

# The axioms of each declaration in a module that compiled (spec 08.5).
{
  while read -r m; do echo "import $m"; done < "$out/$lib.built"
  echo
  awk -F'\t' 'NR==FNR{ok[$1]=1;next} ok[$1]{print "#print axioms " $2}' "$out/$lib.built" "$targets" | sort -u
} > "$out/$lib.axioms.lean"
[ -s "$out/$lib.built" ] && lake env lean "$out/$lib.axioms.lean" > "$out/$lib.axioms.out" 2>&1

# The static checks on each statement (spec 08.4 step 2). An example that compiles shows a possible defect.
{
  while read -r m; do echo "import $m"; done < "$out/$lib.built"
  echo
  echo "set_option maxHeartbeats 40000"
  echo "set_option maxErrors 1000000"
  awk -F'\t' 'NR==FNR{ok[$1]=1;next} ok[$1] && $3=="statement"{print $2}' "$out/$lib.built" "$targets" | sort -u | while read -r d; do
    echo "example : type_of% @$d := by first | trivial | rfl | decide | simp -- om-check $d trivial-goal"
    echo "example : type_of% @$d := by intros; exfalso; first | contradiction | (simp_all; done) | omega | (norm_num at *; done) | linarith -- om-check $d contradictory"
  done
} > "$out/$lib.checks.lean"
[ -s "$out/$lib.built" ] && lake env lean "$out/$lib.checks.lean" > "$out/$lib.checks.out" 2>&1
echo "done $lib"
exit 0
