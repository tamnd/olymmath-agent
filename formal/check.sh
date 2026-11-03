#!/bin/bash
# check.sh LIB TARGETS OUT builds the library LIB of the Lake package in the current directory and runs the checks of spec 08.4 and 08.5 on the declarations in TARGETS. TARGETS has lines of module, declaration and kind, separated by tabs. The outputs go to the directory OUT. om formal record reads them.
set -u
lib=$1
targets=$2
out=$3
mkdir -p "$out"

lake build "$lib" > "$out/$lib.build.log" 2>&1
echo "exit $?" >> "$out/$lib.build.log"

# The modules that compiled are the modules that have an .olean file.
: > "$out/$lib.built"
cut -f1 "$targets" | sort -u | while read -r m; do
  f=".lake/build/lib/lean/$(echo "$m" | tr . /).olean"
  [ -f "$f" ] && echo "$m" >> "$out/$lib.built"
done

# The axioms of each declaration in a module that compiled (spec 08.5).
{
  while read -r m; do echo "import $m"; done < "$out/$lib.built"
  echo
  awk -F'\t' 'NR==FNR{ok[$1]=1;next} ok[$1]{print "#print axioms " $2}' "$out/$lib.built" "$targets" | sort -u
} > "$out/$lib.axioms.lean"
lake env lean "$out/$lib.axioms.lean" > "$out/$lib.axioms.out" 2>&1

# The static checks on each statement (spec 08.4 step 2). An example that compiles shows a possible defect.
{
  while read -r m; do echo "import $m"; done < "$out/$lib.built"
  echo
  echo "set_option maxHeartbeats 40000"
  awk -F'\t' 'NR==FNR{ok[$1]=1;next} ok[$1] && $3=="statement"{print $2}' "$out/$lib.built" "$targets" | sort -u | while read -r d; do
    echo "example : type_of% @$d := by first | trivial | rfl | decide | simp -- om-check $d trivial-goal"
    echo "example : type_of% @$d := by intros; exfalso; first | contradiction | (simp_all; done) | omega | (norm_num at *; done) | linarith -- om-check $d contradictory"
  done
} > "$out/$lib.checks.lean"
lake env lean "$out/$lib.checks.lean" > "$out/$lib.checks.out" 2>&1
echo "done $lib"
