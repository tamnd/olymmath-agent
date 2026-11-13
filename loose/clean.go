// Package loose imports the seed rows that name no position as loose problems (spec 06.8). It cleans the text of each row, drops a row that is not one clear problem, and matches each row with the problems of the content and with the rows before it.
package loose

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/normalize"
)

// Twice is the lowest Jaccard index of the word 5-grams of the two halves of a text that make it one text given twice.
const Twice = 0.6

// MaxText is the longest text, in characters, that the import takes. A longer row is in most cases a whole paper, or a problem with its solution.
const MaxText = 4000

// Reasons why a row is dropped.
const (
	TooShort    = "too short"
	TooLong     = "too long"
	NotStated   = "not a statement"
	Solution    = "solution only"
	Contents    = "table of contents"
	Stub        = "stub"
	Request     = "request"
	Context     = "refers to another problem"
	Figure      = "needs a figure"
	Glued       = "several problems"
	BrokenTeX   = "broken TeX"
	NoLang      = "unknown language"
	WebLink     = "links to a page"
	Template    = "a template"
	Lost        = "formulas lost"
	Science     = "physics or chemistry"
	Economics   = "economics or finance"
	Informatics = "a task of informatics"
	Shortlist   = "a shortlist or longlist row"
	Grading     = "a grading scheme or a note"
	Theorem     = "a theorem of a book"
)

var (
	// headingRE is a heading line at the start of a text, such as "# Task 3." or "## Problem 5", with no formula.
	headingRE = regexp.MustCompile(`^\s*#{1,6}\s*[^\n$]{0,40}\n`)
	// taskHeadRE is the text of a heading that starts the problem, after its problem mark, as in "# 3. Solve the system of equations". Such a heading is kept as the first line of the text, and so is a heading that starts a sentence that the text goes on from, as "# 3. In the equation". Other headings, as "# Task 3. Logic", "## 26. Gold Coins" or "## On the Café Terrace", are titles and are removed.
	taskHeadRE = regexp.MustCompile(`(?i)^(?:solve|find|calculate|compute|evaluate|simplify|prove|show|determine|factor|factorize|plot|sketch|construct|investigate|verify)\b`)
	// exampleRE is the mark of an example of a book in a heading, as "## Example 5. Given the function".
	exampleRE = regexp.MustCompile(`(?i)^example\s*[0-9]+(?:\.[0-9]+)*\.?\s*`)
	// startRE is a heading that starts a sentence, as "In the equation", and goesOnRE the start of a text that goes on from it: a formula or a word in lower case, as "$$x^2+px+q=0$$ the coefficients ...".
	startRE  = regexp.MustCompile(`(?i)^(?:in|on|for|given|let|if|at|from|with|when|it|there|the|a|an)\s+\p{Ll}`)
	goesOnRE = regexp.MustCompile(`^(?:\$|\p{Ll})`)
	// noiseRE is a line that only frames a problem: its points, as "(7 points)" or "## Maximum 15 points"; a heading of the statement, as "## Condition:" or "Condition of the problem"; a request on the form of the answer, as "The answer to the task should be an integer, without units."; or the points at the end, as "Points for the problem: 13.". A lone "#" at the end of a text, left from a heading of the next problem, is a line of it too.
	noiseRE = regexp.MustCompile(`(?im)^[ \t*_]*(?:#+[ \t]*|#*[ \t*_]*(?:\(?\s*(?:maximum\s+|max\.?\s+)?[0-9]+\s*(?:points?|pts?\.?|marks?)\s*\)?|(?:task\s+|problem\s+)?condition(?:\s+of\s+the\s+(?:problem|task))?|the answer (?:to (?:the|this) (?:task|problem) )?should be[^\n]*|points for the (?:problem|task)\s*:?\s*[0-9]+)[ \t.:*_]*)$`)
	// variantRE is the mark of a version of a paper on a line of its own, as "## Variant 1" or "Option 2.". One mark is removed; a text with two holds two versions of the problem.
	variantRE = regexp.MustCompile(`(?im)^[ \t#*_]*(?:variant|option|version)\s*(?:no\.?\s*|№\s*)?[0-9IVX]{1,4}[ \t.:*_]*$`)
	// hashEndRE is a "#" at the end of a text, left from a heading of the next problem.
	hashEndRE = regexp.MustCompile(`\s+#{1,6}\s*$`)
	// blankRE is a run of blank lines that a removed line leaves.
	blankRE = regexp.MustCompile(`\n[ \t]*\n(?:[ \t]*\n)+`)
	// pointsRE is the points of a problem before its text, as "(3 points) Find the distance" or "[4 points] Find the number".
	pointsRE = regexp.MustCompile(`(?i)^\s*[(\[]\s*[0-9]+\s*(?:points?|pts?\.?|marks?)\s*[)\]][\s.:]*`)
	// gradeLeadRE is the grade of a problem before its text, as "(8th grade) Prove" or "(10-11 grades) Find".
	gradeLeadRE = regexp.MustCompile(`(?i)^\s*\(\s*(?:[0-9]{1,2}\s*-\s*)?[0-9]{1,2}\s*(?:th|st|nd|rd)?\s*(?:grades?|class(?:es)?|form)\s*\)[\s.:]*`)
	// sectionRE is a problem number of a section with no point after it, as "10.2 Plot the set of points".
	sectionRE = regexp.MustCompile(`^\s*(?:#{1,6}\s*)?([1-9][0-9]?)(?:\.[0-9]{1,3})+\s+(\p{Lu}(?:\p{Ll}|\s+\p{Ll}))`)
	// gradingRE is a scheme of the points of a solution, or a note of a translator about the text. Neither is a problem.
	gradingRE = regexp.MustCompile(`(?i)\b(?:grading|evaluation|marking|assessment|scoring) (?:criteria|scheme|guidelines|instructions)\b|\bcriteria for (?:evaluation|evaluating|grading)\b|\btranslat(?:or|ion)(?:'s)? notes?\b|\bnote:? the (?:translation|original text)\b`)
	// leadRE is a problem mark at the start of a text, such as "Problem 4.", "**Problem 3.**", "Task 2:", "P6.", "7.", "11.4.", "2)", "Example 3.", "Ex. 119." or the number of a problem of a book, as "1037.", "18.45*." or "(!) 11.".
	leadRE = regexp.MustCompile(`(?i)^\s*(?:\(!\)\s*)?(?:\*\*|__)?\s*(?:(?:problem|task|question|exercise|example|ex\.|задача)\s*-?\s*|[pq]\s*\.?\s*)?([1-9][0-9]{0,3})((?:\.[0-9]{1,3})*)\s*\**\s*(?:\.\)|[.:)])\s*(?:\*\*|__)?(?:\s|$)`)
	// pairLeadRE is the mark of a problem of a grade or a round, as "3-3.", "3-ча 2." or "3-rd 5.", a number of a book that starts with 0, as "0.16.", the mark of an example of a book with no number or with a note, as "Example." or "Example 1 (to item 2).", or the number of a book with the number of an older edition after it, as "112(981)." or "37(!).".
	pairLeadRE = regexp.MustCompile(`(?i)^\s*(?:[1-9][0-9]?-(?:[1-9][0-9]?|(?:ча|cha)\s*[1-9][0-9]?|(?:st|nd|rd|th)\s+[1-9][0-9]?)|0\.[0-9]{1,3}|example(?:\s+[0-9]{1,3})?\s*\([^()\n]{1,40}\)|example|[1-9][0-9]{0,3}\s?\((?:[0-9]{1,4}|!|U)\))\s*[.:]\s+`)
	// wrapRE is a sign that a book repeats at a line break, so the dataset has it twice, as "$y=$ $=\\sqrt{x}$".
	wrapRE = regexp.MustCompile(`=\$[ \n]*\$[ \n]*=|\+\$[ \n]*\$[ \n]*\+|-\$[ \n]*\$[ \n]*-|<\$[ \n]*\$[ \n]*<|>\$[ \n]*\$[ \n]*>|\\times\$[ \n]*\$[ \n]*\\times`)
	// labelLeadRE is the mark of a problem of a round of a national olympiad, a letter and a number, as "B3." of the second round in the Netherlands or "Z2)" of the Slovenian competition.
	labelLeadRE = regexp.MustCompile(`^\s*[A-Z][0-9]{1,2}\s?[.)]\s+`)
	// theoremRE is a theorem of a book at the start of a text, as "Theorem 5.1.2 (Five Color Theorem)" or "Lemma 7.5.2". The book proves it, so it is not a problem.
	theoremRE = regexp.MustCompile(`^(?:Theorem|Lemma|Proposition|Corollary)\s+[0-9]{1,2}(?:\.[0-9]{1,3}){0,2}\b`)
	// replyRE is a message of the translator at the end of a text, as "If you need the entire text in English, please let me know." or "(Note: The last sentence was a mix of French and English. I have corrected it.)".
	replyRE = regexp.MustCompile(`(?i)\bplease let me know\b|\bif you need (?:the )?(?:entire|full|complete|whole) (?:text|translation)\b|\bI have (?:corrected|translated|kept)\b|\bI will provide the (?:English )?translation\b|\boutput the translation result\b|\bkeeping the original text's line breaks\b|\bis the original (?:French|Spanish|German|Chinese|Russian|Croatian|Dutch) text\b`)
	// wordLeadRE is the mark of an example or a problem of a book with no point after its number, as "Example 3 Let", "Exercise 2.4 Find" or "Problem 5 (1994 Shortlist) Prove". A capital letter, a bracket or a formula must follow, so that "Example 5 asks" is kept.
	wordLeadRE = regexp.MustCompile(`^\s*(?:\*\*)?(?i:example|exercise|problem)\s+([1-9][0-9]{0,3})(?:[.-][0-9]{1,3})*(?:\*\*)?\s+(\p{Lu}|\(|\$)`)
	// tightLeadRE is a problem number glued to the text, as "3-There is" or "1.Let".
	tightLeadRE = regexp.MustCompile(`^\s*([1-9][0-9]?)[-.]\p{Lu}\p{Ll}`)
	// markRE is a problem mark inside a text: "Problem 2", "Task 3" or "p2." at the start of a line, or a bold "**3.**" or "**p14.**" anywhere. A row with such a mark after its start holds more than one problem. A post of a whole test often runs its problems on one line, as "... divided by $2024$ .**p14.** A right square pyramid ...".
	markRE = regexp.MustCompile(`(?im)(?:^|\n)\s*(?:\*\*|__)?\s*(?:(?:problem|task|задача)\s*[0-9]{1,2}\b|p[0-9]{1,2}\s*[.:)])|\*\*\s*p?[0-9]{1,2}\s*[.:)]\s*\*\*`)
	// numLineRE is a line that starts with a number and a point, such as "4. Find".
	numLineRE = regexp.MustCompile(`(?m)^\s*([1-9][0-9]{0,3})\.\s`)
	// answerRE is the start of an answer or a solution that a dataset joined to the statement, on a line of its own.
	answerRE = regexp.MustCompile(`(?im)^\s*(?:\*\*|__)?\s*(?:answer|answers|a n s w e r|otbet|ответ|solution|решение|proof|hint|указание|comment)\s*(?:\*\*|__)?\s*[.:]`)
	// solutionRE is a text that starts as a solution, not as a problem.
	solutionRE = regexp.MustCompile(`(?i)^\s*(?:#{1,6}\s*)?(?:[0-9]{1,2}\.\s*)?(?:\*\*|__)?\s*(?:(?:(?:a|an|the)\s+)?(?:my|another|quick|short|simple|alternative|alternate|nice|easy|official|elegant|full|complete)\s+)?(?:solution|sol\.|proof|answer|решение|ответ|otbet)\b`)
	// stepRE is a text that starts with a step of a solution, as "Since the right side is divisible by 49, ..." or "By the condition, ...". Some datasets give the solution of a problem as its statement.
	stepRE = regexp.MustCompile(`^\s*(?:By the condition|From the condition|Since\b|Hence\b|Therefore\b|Thus\b|From this\b|Then\b)`)
	// templateRE is a parameter of a template, as in "Given a regular param1." A template stands for several problems, which a table at its end gives.
	templateRE = regexp.MustCompile(`(?i)\bparam\s?[0-9]+\b`)
	// lostRE is a sentence that a blank line cuts where a dataset dropped a displayed formula, as in "Prove that if the equation\n\nhas three real roots, then", or a note of a translator that says so.
	lostRE = regexp.MustCompile(`(?i)[a-z,]\n\n+(?:where|then|holds?|is|are|has|have|and|for|divides|takes|to|gdzie|dla|oraz|jest|ma)\b|\b(?:not (?:provided|given|included) in the (?:original )?text|original text had|untranslated)\b`)
	// lostRE2 is a sentence that stops at a word that brings in a formula, as "satisfy the condition", with a blank line and a new sentence after it: the formula was lost. A note of a translator, as "The translation is provided as requested", stands in its place in some rows.
	lostRE2 = regexp.MustCompile(`\b(?:condition|conditions|equation|equations|inequality|inequalities|equality|equalities|identity|identities|formula|formulas|relation|relations|relationship|system|expression|function|functions|holds?|satisfied|true|following|defined|then|that|if|by|number|numbers|polynomial|sequence|which|where|arranged|table|product|sum|and)\s*\n\n+[A-Z(]`)
	// lostEndRE is a text that stops at a word that brings in a formula or a claim, as "Prove that" or "satisfying the equation", or at a colon: the formula or the claim was lost.
	lostEndRE = regexp.MustCompile(`(?i)(?:\b(?:that|then|and|if|where|equation|equations|inequality|inequalities|identity|equality|condition|conditions|system|holds|following)|:)[\s#]*$`)
	// stepWordRE is a word of the steps of a solution. A text with two or more of them is a solution, not a problem: "Then its discriminant is ..., therefore ...". A problem may use one, as "Prove that then ...", but in the fourth batch nearly every text with two of them was a solution.
	stepWordRE = regexp.MustCompile(`(?i)\b(?:we find|we will show|we prove|let(?:'s| us) (?:show|prove|find|consider|denote|check|note|estimate|compute|calculate)|indeed|hence|therefore|thus|it follows|consequently|which means|this means|as required|q\.e\.d|substituting|it remains to|from (?:here|this|where)|then its|so the answer|the answer is)\b`)
	// economicsRE is a term of economics or of personal finance, out of scope as the other sciences: some olympiad sets mix the problems of an economics or a financial literacy olympiad with those of mathematics.
	economicsRE = regexp.MustCompile(`(?i)perfectly competitive|demand function|supply function|monetary units|microeconomics|macroeconomics|\beconomics\b|taxpayer|tax resident|tax deduction|income tax|securities|financial instrument|\bScore: [0-9]|\b(?:mortgage|inflation|interest-free loan|loan agreement|nominal value|cadastral|tax (?:rate|period|return|base)s?|(?:transport|land|property) tax|pension (?:fund|savings|programs?)|(?:funded|insurance) part of (?:her|his|the) pension|insurance premium|brokerage (?:account|firm)|stock market|mutual fund|labou?r market|alternative costs?|opportunity costs?|full score|financial (?:institution|organi[sz]ation|literacy|situation)|credit card|real return|exchange rate of|maximi[sz]ing (?:his|its|their) profit|budget revenues?|zero-coupon|ordinary shares|preferred shares|investment options?|deposit rates?|loyalty program|bonus program|producers? surplus|consumer surplus)\b`)
	// noteRE is a note of the translator at the end of the text about its form, as "Note: The translation maintains the original format and line breaks.". The note is cut and the problem is kept, unless the note says that a part of the text is missing or unclear (doubtRE); such a row is dropped.
	noteRE  = regexp.MustCompile(`(?is)\s*(?:-{3,}\s*)?[(*_]*\s*(?:Note:?\s*)?The (?:provided )?translation (?:maintains|preserves|keeps|retains|is provided)\b.*$`)
	doubtRE = regexp.MustCompile(`(?i)\bhowever\b|\bnot\b|\bmissing\b|\bincomplete\b|\bunclear\b|\bseems?\b|\blost\b`)
	// promptRE is a request to translate that a dataset left at the end of the text.
	promptRE = regexp.MustCompile(`(?is)\s*(?:Translate the (?:text above|above text) into English\b|将上面的文本翻译成英文|翻译结果如上).*$`)
	// scienceRE is a term of physics or chemistry. Problems of physics and of the other sciences are out of scope (spec 02); some olympiad sets mix them with the problems of mathematics. A puzzle that only names physics, as a club of physics, does not match.
	scienceRE = regexp.MustCompile(`(?i)\((?:physics|chemistry)\)|\b(?:refractive|refraction|coefficient of friction|friction(?:al)? force|voltage|resistors?|resistivity|internal resistance|capacitors?|capacitance|piston|heat capacity|specific heat|latent heat|buoyan\w*|kinetic energy|potential energy|pendulum|wavelength|joules?|ohms?|electronegativity|hybridi[sz]ed|hybridization|moles? of|molar mass|(?:sulfuric|sulphuric|nitric|hydrochloric|hydroiodic|orthophosphoric|phosphoric|acetic) acid|isotopes?|acceleration due to gravity|air resistance|thermostat|spring constant|stiffness(?:es)? of the springs?|amperes?|ammeter|voltmeter|galvanic|emf|electromotive|ideal gas|(?:converging|diverging|collecting|thin) lens|focal length|optical (?:power|axis|center|centre)|electric (?:field|current|charge|circuit)|electrical circuit|current source|precipitat\w*|aqueous|alkanes?|alkynes?|alkenes?|gravitational forces?|elastic collision|heat engines?|heaters?|density of (?:the |a |its )?(?:water|wood|ice|kerosene|aluminum|aluminium|copper|iron|steel|oil|material|body|liquid|mercury|glass|air|saltwater|milk)|densit(?:y|ies) (?:is|are) defined|spring scales?|on a spring|of the spring|DC power|power source|boiling point|monatomic|diatomic|thermally insulated|resistance force|calorimeter|eudiometer|jet engine|sound signal|(?:plane|ideal|flat) mirror|refrigerat(?:ing|ion) machines?|hot air balloon|charged (?:with|to)|carbon dioxide|chemical (?:reaction|experiment)s?|fully submerged|completely submerged in (?:water|kerosene|a liquid|liquid)|floats in (?:a |the )?(?:water|lake|liquid|glass|vessel))\b|kg\s*/\s*m\b`)
	// informaticsRE marks a task of an olympiad of informatics or of an applied contest: a heading such as "Text of the task." or "Problem statement.", a request to write a program, or terms of programming. Some olympiad sets mix such tasks with the problems of mathematics.
	informaticsRE = regexp.MustCompile(`\b(?:Text of the task|Task text|Task [Dd]escription|Problem text|Problem statement)\s*[.:]|(?i)\bwrite a program (?:that|which) (?:finds|outputs|prints|reads)\b|\((?:informatics|computer science)\)|\b(?:data analyst|input data|output data|time limit|memory limit|system administration|information security)\b`)
	// shortlistRE is the mark of a problem of the IMO shortlist or longlist in a compendium: the country that proposed it, as "(GBR 2)" or "A2 (GBR)" at the start, or a note that the problem was set at the IMO, as "${ }^{\mathrm{IMO} 5}$". The row does not give the year, and the content has the shortlists, the longlists and the IMO by position, so such a row is left to them.
	shortlistRE = regexp.MustCompile(`^\s*(?:[IVX]{1,3}\s+[0-9]{1,2}\s+)?(?:[ACGNS][0-9]{1,2}\s+)?\([A-Z]{3}(?:\s+[0-9]{1,2})?\)|\^\{\s*\\(?:mathrm|text)\s*\{\s*IMO`)
	// personRE is a line of a poster about himself. With no formula, no question and no task it marks a forum post that is news and not a problem.
	personRE = regexp.MustCompile(`\bI(?:'m| am| have|'ve| will|'ll| plan)\b|\bsorry\b`)
	// taskRE is a word that asks for something or states a claim. A text with none of them, once a cut leaves it or with no formula, is not a problem: "Given an odd number $m$." is what is left of "Given an odd number $m$. Proof: for any $c$ ...", where Proof stands for Prove.
	taskRE = regexp.MustCompile(`(?i)\?|\b(?:prove|proof|show|find|determine|compute|calculate|evaluate|solve|what|how|which|whether|is there|are there|does|do|can|could|must|exists?|construct|describe|decide|give|simplify|express|count|demonstrate|verify|then|if|when|holds?|restore|reconstruct|mark|cut|inscribe|transform|draw|split|divide|arrange|fill|colou?r|obtain|define|consider|propose|discuss|build|design|let)\b|_{3,}|\b(?:докаж|найд|определ|сколько|можно|верно)`)
	// contentsRE finds the lines of a table of contents: a dotted leader or the word contents.
	contentsRE = regexp.MustCompile(`(?i)\.{5,}|\\ldots\s*\\ldots\s*\\ldots|\btable of contents\b|^\s*contents\s*$`)
	// stubRE names a fact of the text around the row, such as "Prove the lemma." or "Prove Claim 2.". A row with such a name, no formula and few words is a stub.
	stubRE = regexp.MustCompile(`(?i)\b(?:lemma|claim|corollary|proposition)\b`)
	// requestRE is a post that asks for help or for a translation.
	requestRE = regexp.MustCompile(`(?i)\b(?:please|pls|plz)\b[^.?!]{0,40}\b(?:help|translate|solve|post|check)\b|\bhelp me\b|\bappreciate (?:any |some )?help\b|\bcan (?:someone|anyone|somebody)\b|\b(?:anyone|somebody|someone) (?:can|could|help|knows?)\b|\bany (?:hints?|ideas?|help)\b|\bthank(?:s| you)\b|\btranslat(?:e|ion) (?:this|please|needed|request)|\bi (?:need|want) (?:a |the )?(?:solution|help|hint)|\bhow (?:do|can|should) i\b|\bi (?:can't|cannot|couldn't|could not) (?:solve|prove|find)\b|\bmy (?:teacher|friend|solution|attempt)\b`)
	// refRE is a text of a book that names another problem or a section, as "(see problem II. 296)", "Solve problem 1.67 using ..." or "See also problems 248 a) - c) from § 1". Such a text depends on the book around it. The mark of a problem of the Polish olympiad at the start, as "XXXVIII OM - I - Problem 11", is not a reference.
	refRE = regexp.MustCompile(`(?i)\b(?:problems?|exercises?)\s+(?:[IVX]{1,3}\.\s*)?[0-9]{1,4}(?:\.[0-9]{1,3})?\b|§|^\s*the same (?:question|problem)\b`)
	// verbRE is a word that asks for a computation, as "Multiply:" or "Differentiate".
	verbRE = regexp.MustCompile(`(?i)\b(?:multiply|divide|add|subtract|expand|factori[sz]e|differentiate|integrate|reduce|graph|draw|estimate|compare|check|rationali[sz]e|decompose|transform)\b`)
	// valueRE is a formula that ends in "=", which asks for its value, as "$1^{3}+2^{3}+\\ldots+9^{3}=$".
	valueRE = regexp.MustCompile(`=\s*(?:\\quad\s*)?(?:\\ldots|\\cdots|\\_+|_+)?\s*\$+\s*[.,]?\s*$|=\s*\$+\s*,`)
	// tailRE is a line that a book puts after a problem: a heading of the next section in capitals, as "TILINGS WITH POLYGONS", or a row of stars.
	tailRE = regexp.MustCompile(`(?:\n\s*(?:\$\$\s*)?\*\s*\*\s*\*(?:\s*\$\$)?|\n\s*\n\s*\p{Lu}{2,}(?:[ \-]+\p{Lu}{2,})+)\s*$`)
	// olympiadTailRE is the heading of an olympiad and the table of its problems by class and day that a book puts after the last problem of the olympiad, as "17th All-Union Olympiad, 1983 (Kishinev)" and "| Class | 1st day | 2nd day |". The heading has a year and no point, colon or question mark, so a table that a problem gives is kept.
	olympiadTailRE = regexp.MustCompile(`(?is)\n\s*\n(?:\$\$\s*\\begin\{gathered\}\s*)?[^\n]*\bolympiad\b[^.:?]*?\b(?:19|20)[0-9]{2}\b[^.:?]*?\n\s*\n\s*\|\s*(?:class|racc|pass|grade)\b.*$`)
	// roundTailRE is the note on the next round that the Czech and Slovak olympiad prints after the last problem of a round, as "The regional round of category B takes place on Tuesday, January 22, 2008 so that ..." or "School - written part of the 1st round, category C is held", with the rules of the round (the time, the points, the aids) after it. The note is in a paragraph of its own after the problem and names a category, so it is cut with everything after it. The verb ends the line or a date follows it, so a sentence of a problem such as "is held to solve" stays.
	roundTailRE = regexp.MustCompile(`(?is)\n\s*\n[^\n?]{0,90}\bcategory\b[^\n?]{0,30}?\b(?:takes|will take|is|will be) (?:place|held)(?:[ \t]*(?:\n|$)|\s+on\b|,).*$`)
	// slashFracRE is a fraction that the dataset wrote as a superscript, a slash and a subscript, as "{ }^{x} /{ }_{10}" for x/10, and slashTopRE the same with no subscript, as "{ }^{1} / m".
	slashFracRE = regexp.MustCompile(`\{ \}\^\{([^{}]*)\}\s?/\s?\{ \}_\{([^{}]*)\}`)
	slashTopRE  = regexp.MustCompile(`\{ \}\^\{([^{}]*)\}\s?/\s?`)
	// pageRE is the number of a page that a book prints after the last problem of the page, as "388", alone in the last paragraph of a text. A number of four digits is most often a year, so it stays.
	pageRE = regexp.MustCompile(`^[0-9]{1,3}$`)
	// gradeRE is the grade or the points of a problem in the last paragraph of a text, as "(10-11 grades)", "(10th grade)" or "(7-8 pts)", once its "$" signs, "\\text" and braces are taken out.
	gradeRE = regexp.MustCompile(`(?i)^\((?:[0-9]{1,2}-)?[0-9]{1,2}(?:(?:th|st|nd|rd)(?:grades?|classes?|form)?|grades?|classes?|pts\.?|points)\)\.?$`)
	// gradeSkipRE is what is taken out of a last paragraph before gradeRE reads it.
	gradeSkipRE = regexp.MustCompile(`\$|\\text|[{}\s]`)
	// contextRE is a text that depends on another problem.
	contextRE = regexp.MustCompile(`(?i)\b(?:previous|preceding|above|last|prior) (?:problem|exercise|question|task)\b|\b(?:problem|exercise|task) [0-9]{1,3} (?:above|before)\b|\bas in (?:problem|exercise|task) [0-9]{1,3}\b|\bfrom (?:problem|exercise|task) [0-9]{1,3}\b|\bof (?:the )?(?:previous|preceding) part\b`)
	// texTagRE is the TeX that a Russian problem book wraps around its topic tags, as in "$\left[\begin{array}{l}\text { Invariants }\end{array}\right]$". It is removed only to read the line.
	texTagRE = regexp.MustCompile(`\\(?:begin|end)\{(?:array|aligned|gathered)\}(?:\{[lcr]*\})?|\\(?:text|underline|mathrm|mathbf|textbf|frac)\b|\\(?:left|right)(?:\\[{}]|[.\[\]()|])?|\\\\|[&${}]`)
	// tagLineRE is a line of topic tags, as "[ Sine theorem ] [ Auxiliary circle ]" or "[Mathematical logic (miscellaneous).] Complex".
	tagLineRE = regexp.MustCompile(`^\[\s*(?:\p{Lu}|\[)|^\[\s*\p{L}.*\s.*\]$`)
	// leadTagsRE is topic tags in brackets on the first line of the problem, as "[Equilateral Triangle] Prove that".
	leadTagsRE = regexp.MustCompile(`^(?:\[\s*\p{Lu}[^\[\]\n?=]{0,150}\]\s*)+`)
	// lettersRE is a word of three letters or more, and lowerRE one of three small letters, which a tag has and "[x]" or "[ABC]" has not.
	lettersRE = regexp.MustCompile(`\p{L}{3}`)
	lowerRE   = regexp.MustCompile(`\p{Ll}{3}`)
	// taskWordRE is a word that asks for something, which a topic tag does not have.
	taskWordRE = regexp.MustCompile(`(?i)\b(?:solve|find|calculate|compute|evaluate|simplify|prove|show|determine)\b`)
	// helpsRE is the end of a tag such as "Symmetry helps solve the problem".
	helpsRE = regexp.MustCompile(`(?i)\bhelps? (?:to )?solve\b`)
	// tableMarkRE is the difficulty or the number of the problem in a table of tags, as "Difficulty: 3" or "Problem 109870 Topics".
	tableMarkRE = regexp.MustCompile(`(?i)\bdifficulty\s*:\s*[0-9+-]+|\bproblem\s+[0-9]+\s+(?:topics|themes)\b`)
	// groupRE is a group in brackets.
	groupRE = regexp.MustCompile(`\[[^\[\]]*\]`)
	// leadTexTagRE is topic tags wrapped in TeX on the first line of the problem, as "$\left.\begin{array}{l}\text { Cone }\end{array}\right]$ Three cones".
	leadTexTagRE = regexp.MustCompile(`^\$[^$?=]*\\text[^$?=]*\$\s*`)
	// tableLineRE is a line of a table of topic tags, with the grades before it, as "10,11 |" and "| :---: | :---: |".
	tableLineRE = regexp.MustCompile(`^(?:[0-9]{1,2}\s*,\s*)*(?:[0-9]{1,2}\s*,?\s*)?\|`)
	// gradeLineRE is the grades or the mark of difficulty of a problem on a line of its own, as "8,9", "9,10,11" or "$3+$".
	gradeLineRE = regexp.MustCompile(`^(?:[0-9]{1,2}\s*,\s*)*[0-9]{1,2}\s*[,+-]?$`)
	// authorLineRE is a line of the authors, as "Shapovalov A.V.", "Kanel-Belov A.Ya.", "X. Khanov" or "Authors: Netay I.V., Baranov D.V.", or the mark "Folklore".
	authorLineRE = regexp.MustCompile(`^(?:(?i:authors?)\s*:.*|(?i:folklore|folkior|folklor)\.?|(?:(?:\p{Lu}[\p{Ll}'-]+|\p{L}{1,2}\.|von|der|and)[\s,.:;]*)+)$`)
	// initialRE is an initial of a name, as "A." in "Shapovalov A.V.".
	initialRE = regexp.MustCompile(`(?:^|[\s.])\p{L}{1,2}\.`)
	// leadTeXRE is the TeX that bookLead removes to read a paragraph before a problem.
	leadTeXRE = regexp.MustCompile(`\\(?:begin|end)\{(?:array|aligned|gathered)\}(?:\{[lcr]*\})?|\\(?:text|underline|mathrm|mathbf|mathbb|textbf|stackrel|left|right|frac|cdot|quad|section\*?)\b|\\\\|[&${}~*]`)
	// leadAuthorRE is an author line that the translation garbled, as "Avtor: Shapovalov A.V." or "Aвmp Zhenodarov R.G.".
	leadAuthorRE = regexp.MustCompile(`(?i)^\W{0,4}(?:authors?|avor|auto|avt|avtor|aвmp|avoo|aesm|aamop)\b`)
	// leadGuardRE is the start of a paragraph that is the problem itself: a word that asks for something, a verb or an article.
	leadGuardRE = regexp.MustCompile(`(?i)^\W*(?:given|let|find|prove|solve|show|compute|calculate|evaluate|simplify|determine|construct|consider|suppose|denote|is|are|can|does|do|how|what|which|who|why|in|on|for|if|from|at|two|three|four|five|a|an|the|one|there|answer|example|problem|task|exercise|part|case)\b`)
	// leadMarkRE is a mark of difficulty or of grades on a paragraph of its own, as "3+", "8,9" or "4 [ Parity ]".
	leadMarkRE = regexp.MustCompile(`^(?:[1-6][+-]?|[0-9]{1,2}(?:[,.][0-9]{1,2})+)\s*(?:\[[^?=]*|\|.*)?$`)
	// leadBracketRE is a paragraph that is only a topic tag in brackets, or a lone bracket.
	leadBracketRE = regexp.MustCompile(`^\[[^?=]*\]$|^[\[\]]$`)
	// leadNameRE is a name that the translation garbled, as "Crackov d:", "Tokarev S.I" or "$\Phi$ renkin B.R.".
	leadNameRE = regexp.MustCompile(`^[^?=]{1,40}\b[A-Za-z]?\.?\s?[Dd]:$|^[A-Z][a-z]+ [A-Z]\.\s?[A-Z]\.?$|^[^?=]{0,30}\b[A-Z]\.\s?[A-Z]\.$`)
	// leadKeepRE is a paragraph that starts a formula or a part of the problem, as "$$", "a)" or "2.".
	leadKeepRE = regexp.MustCompile(`^\s*(?:\$\$|[a-z]\)|[0-9]+\)|[0-9]+\.)`)
	// leadNextRE is the start of the paragraph of a problem after a line of the book.
	leadNextRE = regexp.MustCompile(`^[A-Z$\[]`)
	// leadTextRE is a \text group, and leadMathRE an inline formula, which bookLead leaves out when it counts the words of a paragraph.
	leadTextRE = regexp.MustCompile(`\\text\s*\{[^{}]*\}`)
	leadMathRE = regexp.MustCompile(`\$[^$]*\$`)
	// leadWordRE is a word of three small letters or more.
	leadWordRE = regexp.MustCompile(`\b[a-z]{3,}\b`)
	// leadTagsOnlyRE is what is left of a block of topic tags once its \text groups and its TeX are taken out.
	leadTagsOnlyRE = regexp.MustCompile(`^[\s\[\]().,:;|]*$`)
	// escapedRE is a dollar sign that the dataset escaped where it meant a formula, as "$\$ A B C \$$" or "\$x\$". The TeX of such a row is broken, and in about half of them the names of points or the formulas are lost. An escaped dollar before a digit is money, as in "\$5", and is kept.
	escapedRE = regexp.MustCompile(`\\\$(?:[^0-9]|$)`)
	// figureRE is a text that needs a figure that the row does not have.
	figureRE = regexp.MustCompile(`(?i)\[asy\]|\bfigure shown\b|\bas shown\b|\bthe (?:diagram|picture|photo)\b|\bphoto\b|\(see (?:the )?fig|\bfig\.\s*[0-9]|\[img\]|!\[|<img|\\includegraphics|https?://\S+\.(?:png|jpe?g|gif|svg)\b|\b(?:see|in|on) (?:the )?(?:figure|picture|diagram|drawing|image)\b|\b(?:figure|picture|diagram|drawing) (?:below|above|shows|on the right|on the left)\b|\bshown (?:below|above|in the (?:figure|picture|diagram))\b|\bpictured\b|\b(?:рисун\p{L}*|рис\.)`)
)

// Clean gives the text of a row as a statement, or "" and the reason why the row is not one clear problem. The text is normalized (spec 06.4), a heading line and a problem mark at the start are removed, and an answer or a solution that the dataset joined to the end is cut.
func Clean(raw string) (string, string) {
	plain := promptRE.ReplaceAllString(markup(raw), "")
	if m := noteRE.FindStringIndex(plain); m != nil && !doubtRE.MatchString(plain[m[0]:]) {
		plain = plain[:m[0]]
	}
	if len(variantRE.FindAllStringIndex(plain, 2)) > 1 {
		return "", Glued
	}
	plain = variantRE.ReplaceAllString(plain, "")
	plain = noiseRE.ReplaceAllString(plain, "")
	plain = strings.TrimSpace(blankRE.ReplaceAllString(hashEndRE.ReplaceAllString(plain, ""), "\n\n"))
	text := wrapRE.ReplaceAllStringFunc(normalize.Statement(plain, ""), func(m string) string { return m[:strings.IndexByte(m, '$')] })
	// cut tells if an answer or a solution was cut from the end of the text.
	cut := len(normalize.CutTail(plain)) < len(strings.TrimSpace(plain))
	text = bookHead(text)
	if solutionRE.MatchString(text) {
		return "", Solution
	}
	if m := headingRE.FindStringIndex(text); m != nil && !strings.ContainsAny(text[:m[1]], "?") {
		head := strings.TrimSpace(strings.TrimLeft(text[:m[1]], "# \t"))
		if l := leadRE.FindStringIndex(head); l != nil {
			head = strings.TrimSpace(head[l[1]:])
		}
		head = strings.TrimSpace(pointsRE.ReplaceAllString(exampleRE.ReplaceAllString(head, ""), ""))
		rest := strings.TrimSpace(text[m[1]:])
		if taskHeadRE.MatchString(head) || startRE.MatchString(head) && !strings.ContainsAny(head[len(head)-1:], ".!:)]") && goesOnRE.MatchString(rest) {
			text = head + "\n\n" + rest
		} else {
			text = rest
		}
	}
	text = strings.TrimSpace(pointsRE.ReplaceAllString(pairLeadRE.ReplaceAllString(labelLeadRE.ReplaceAllString(text, ""), ""), ""))
	text = cutReply(text)
	lead := 0
	if m := leadRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = strings.TrimSpace(text[m[1]:])
	} else if m := tightLeadRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = strings.TrimSpace(text[m[3]+1:])
	} else if m := wordLeadRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = text[m[4]:]
	} else if m := sectionRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = text[m[4]:]
	}
	text = strings.TrimSpace(gradeLeadRE.ReplaceAllString(pointsRE.ReplaceAllString(text, ""), ""))
	text = bookTail(strings.TrimSpace(tailRE.ReplaceAllString(olympiadTailRE.ReplaceAllString(roundTailRE.ReplaceAllString(firstCopy(text), ""), ""), "")))
	text = slashTopRE.ReplaceAllString(slashFracRE.ReplaceAllString(text, "$1 / $2"), "$1 / ")
	text = bookLead(text)
	// A step of a solution and the mark of the compendium often follow a problem number, as in "5. Since ..." or "27. (BLR) Ten points ...", so they are read after the number is cut.
	switch {
	case stepRE.MatchString(text):
		return "", Solution
	case shortlistRE.MatchString(text):
		return "", Shortlist
	case theoremRE.MatchString(text):
		return "", Theorem
	}
	if m := answerRE.FindStringIndex(text); m != nil {
		if m[0] < canon.MinText {
			return "", Solution
		}
		text = strings.TrimSpace(text[:m[0]])
		cut = true
	}
	if markRE.MatchString(text) {
		return "", Glued
	}
	if lead > 0 {
		// A row that starts with problem 3 and has a line that starts with 4. holds the next problem too.
		for _, m := range numLineRE.FindAllStringSubmatch(text, -1) {
			if n, _ := strconv.Atoi(m[1]); n == lead+1 {
				return "", Glued
			}
		}
	}
	n := utf8.RuneCountInString(text)
	switch {
	case n < canon.MinText:
		return "", TooShort
	case n > MaxText:
		return "", TooLong
	case contentsRE.MatchString(text):
		return "", Contents
	case stubRE.MatchString(text) && !strings.Contains(text, "$") && n < 150:
		return "", Stub
	case requestRE.MatchString(text):
		return "", Request
	case contextRE.MatchString(text), refRE.MatchString(polishRE.ReplaceAllString(text, "")):
		return "", Context
	case templateRE.MatchString(text):
		return "", Template
	case lostRE.MatchString(text), lostRE2.MatchString(text), lostEndRE.MatchString(text):
		return "", Lost
	case scienceRE.MatchString(text):
		return "", Science
	case economicsRE.MatchString(text):
		return "", Economics
	case gradingRE.MatchString(text):
		return "", Grading
	case informaticsRE.MatchString(text):
		return "", Informatics
	case stepWords(text) >= 2:
		return "", Solution
	case figureRE.MatchString(text):
		return "", Figure
	case linkRE.MatchString(text):
		return "", WebLink
	case !canon.IsStatement(text), cut && !taskRE.MatchString(text), bare(text), !strings.ContainsAny(text, "$?") && personRE.MatchString(text) && !taskRE.MatchString(text):
		return "", NotStated
	case !balanced(text), escapedRE.MatchString(text):
		return "", BrokenTeX
	}
	return text, ""
}

// bookTail cuts the number of a page and the grade or the points of the problem that a book prints after it, each in a last paragraph of its own. A page number is cut only after a text that ends as a sentence or a formula ends, so that "Find the number:\n\n17" keeps its number.
func bookTail(text string) string {
	for {
		k := strings.LastIndex(text, "\n\n")
		if k < 0 {
			return text
		}
		head, last := strings.TrimSpace(text[:k]), strings.TrimSpace(text[k+2:])
		switch {
		case pageRE.MatchString(last) && head != "" && strings.ContainsAny(head[len(head)-1:], ".?!$)"):
		case gradeRE.MatchString(gradeSkipRE.ReplaceAllString(last, "")):
		default:
			return text
		}
		text = head
	}
}

// bookHead cuts the lines that a Russian problem book prints before a problem: its topic tags, in brackets or in a table and often wrapped in TeX, its grades, its authors and the mark "Folklore". A line is cut only when it has no question mark and no equals sign and a line of the problem comes after it. A table is cut whole or not at all, and only when it has no digits, so a table of the problem itself is kept.
func bookHead(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		n := 1
		switch {
		case line == "":
		case tableLineRE.MatchString(texTagRE.ReplaceAllString(line, " ")):
			for i+n < len(lines) && strings.Contains(lines[i+n], "|") {
				n++
			}
			if !tagTable(lines[i : i+n]) {
				n = 0
			}
		case !headLine(line):
			n = 0
		}
		if n == 0 || strings.TrimSpace(strings.Join(lines[i+n:], "\n")) == "" {
			break
		}
		i += n
	}
	rest := strings.TrimSpace(strings.Join(lines[i:], "\n"))
	m := leadTagsRE.FindString(rest)
	if m == "" || !lowerRE.MatchString(m) {
		m = leadTexTagRE.FindString(rest)
		if !strings.ContainsAny(m, "[]") {
			m = ""
		}
	}
	if line, _, _ := strings.Cut(rest, "\n"); m == "" && strings.Count(line, "|") == 1 && strings.HasPrefix(line, "|") {
		m = "|"
	}
	if r, _ := utf8.DecodeRuneInString(strings.TrimSpace(rest[len(m):])); m != "" && (r == '$' || unicode.IsUpper(r)) {
		rest = strings.TrimSpace(rest[len(m):])
	}
	return rest
}

// bookLead cuts up to four paragraphs that a problem book prints before a problem and that bookHead does not read line by line: an author line that the translation garbled, a mark of difficulty such as "3+", a block of topic tags in a formula, a topic tag in brackets, a garbled name and a short line of one word or none, as a name, or of no word when it has a formula. A paragraph is cut only when it has no question mark and no equals sign and a paragraph comes after it.
func bookLead(text string) string {
	ps := strings.Split(text, "\n\n")
	i := 0
	for i < len(ps)-1 && i < 4 && leadLine(strings.TrimSpace(ps[i]), strings.TrimSpace(ps[i+1])) {
		i++
	}
	return strings.TrimSpace(strings.Join(ps[i:], "\n\n"))
}

// leadLine tells if a paragraph p before the paragraph next is a line of a book that is not the problem (see bookLead).
func leadLine(p, next string) bool {
	// The mark of the Polish olympiad is read after Clean, so it stays.
	if strings.ContainsAny(p, "?=") || utf8.RuneCountInString(p) > 400 || polishRE.MatchString(p) {
		return false
	}
	if leadTags(p) {
		return true
	}
	plain := strings.Join(strings.Fields(leadTeXRE.ReplaceAllString(p, " ")), " ")
	// A paragraph with a formula and a word is in most cases a sentence of the problem, as "$P(x)$ is a polynomial.".
	words, most := leadWords(p), 1
	if strings.Contains(p, "$") {
		most = 0
	}
	switch {
	case leadAuthorRE.MatchString(plain) && utf8.RuneCountInString(plain) < 110:
		return true
	case strings.HasPrefix(plain, "##") && strings.ContainsAny(plain, "[]"):
		return true
	case leadMarkRE.MatchString(plain), leadBracketRE.MatchString(plain) && words <= 12, leadNameRE.MatchString(plain):
		return true
	case utf8.RuneCountInString(p) <= 60 && words <= most && !leadGuardRE.MatchString(plain) && !leadKeepRE.MatchString(p) && !strings.HasSuffix(p, ":") && leadNextRE.MatchString(next):
		// One word before a formula is the start of the problem, as "Solve\n\n$$x^2=4$$".
		return len(strings.Fields(p)) > 1 || !strings.HasPrefix(next, "$") || strings.HasSuffix(p, ".")
	}
	return false
}

// leadTags tells if a paragraph is only topic tags in \text groups with brackets, wrapped in a formula, as "$$\n\\text { [ Invariants ] }\n$$".
func leadTags(p string) bool {
	wrapped := strings.HasPrefix(p, "$$") && strings.HasSuffix(p, "$$") || strings.HasPrefix(p, "\\")
	if !wrapped || !strings.Contains(p, "\\text") || !strings.Contains(p, "[") {
		return false
	}
	return leadTagsOnlyRE.MatchString(leadTeXRE.ReplaceAllString(leadTextRE.ReplaceAllString(p, ""), " "))
}

// leadWords counts the words of three small letters or more of a paragraph outside its formulas and its \text groups.
func leadWords(p string) int {
	return len(leadWordRE.FindAllString(leadMathRE.ReplaceAllString(leadTextRE.ReplaceAllString(p, ""), ""), -1))
}

// tagTable tells if the lines of a table at the start of a text are a table of topic tags: it has no digits but the grades before it, no question mark and no equals sign.
func tagTable(lines []string) bool {
	all := tableMarkRE.ReplaceAllString(tableLineRE.ReplaceAllString(texTagRE.ReplaceAllString(strings.Join(lines, "\n"), " "), "|"), "")
	return !strings.ContainsAny(all, "0123456789?=")
}

// headLine tells if a line before the text of a problem is a line of a book that is not the problem: a topic tag, the grades, an author or "Folklore".
func headLine(line string) bool {
	if strings.ContainsAny(line, "?=") || utf8.RuneCountInString(line) > 300 || strings.Contains(line, "](") {
		return false
	}
	plain := strings.TrimSpace(texTagRE.ReplaceAllString(line, " "))
	switch {
	case tagLineRE.MatchString(plain) && lettersRE.MatchString(plain):
		// The words out of the brackets are a tag that lost its bracket or a mark such as "Complex", and not a sentence of the problem.
		left := groupRE.ReplaceAllString(plain, "")
		return !taskWordRE.MatchString(helpsRE.ReplaceAllString(left, "")) && (utf8.RuneCountInString(left) <= 80 || !strings.ContainsAny(left, "0123456789"))
	case line == "[", strings.HasPrefix(line, "$") && strings.Contains(line, "\\text") && strings.ContainsAny(line, "[]"):
		return true
	case gradeLineRE.MatchString(plain):
		return true
	case authorLineRE.MatchString(plain) && utf8.RuneCountInString(plain) <= 120:
		low := strings.ToLower(plain)
		return strings.HasPrefix(low, "author") || strings.HasPrefix(low, "folk") || initialRE.MatchString(plain) && !taskHeadRE.MatchString(plain)
	}
	return false
}

// bare tells if a text is only a formula, with fewer than three words around it and none that asks for something, as "$f(x)=\\frac{x^{3}}{x^{2}-2}$.": the task was in a heading that the dataset left out. A formula that ends in "=" or "=?" asks for its value, and a list of choices makes a question, so neither is bare.
func bare(text string) bool {
	prose := strings.TrimSpace(stripMath(text))
	return proseWords(text) < 3 && !taskRE.MatchString(prose) && !taskHeadRE.MatchString(prose) && !verbRE.MatchString(prose) && !strings.Contains(text, "?") && !strings.Contains(text, "(A)") && !valueRE.MatchString(text)
}

// proseWords counts the words of a text outside its formulas. A text such as "$f(x)=\\frac{x^{3}}{x^{2}-2}$." has none: the task was in a heading that the dataset left out.
func proseWords(text string) int {
	n := 0
	for _, w := range strings.Fields(stripMath(text)) {
		if utf8.RuneCountInString(strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) })) >= 2 {
			n++
		}
	}
	return n
}

// balanced tells if the dollar signs of a text that are not escaped pair up, and if the braces inside the formulas pair up.
func balanced(text string) bool {
	dollars, depth := 0, 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '$':
			dollars++
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return dollars%2 == 0 && depth == 0
}

// languages maps the names of languages that the datasets give to their codes.
var languages = map[string]string{
	"english": "en", "en": "en", "spanish": "es", "español": "es", "es": "es", "russian": "ru", "ru": "ru",
	"chinese": "zh", "chinese (traditional)": "zh", "zh": "zh", "mongolian": "mn", "macedonian": "mk",
	"german": "de", "de": "de", "vietnamese": "vi", "vi": "vi", "português": "pt", "portuguese": "pt", "pt": "pt",
	"french": "fr", "fr": "fr", "arabic": "ar", "afrikaans": "af",
}

// stopWords are common words of the languages that Lang tells apart by their words.
var stopWords = map[string][]string{
	"en": {"the", "of", "and", "that", "is", "are", "be", "let", "find", "prove", "all", "with", "for", "in"},
	"es": {"el", "la", "los", "las", "de", "que", "es", "un", "una", "sea", "con", "para", "por", "y", "en"},
	"pt": {"o", "os", "as", "de", "que", "um", "uma", "seja", "com", "para", "por", "e", "não", "são"},
	"fr": {"le", "la", "les", "de", "des", "que", "est", "un", "une", "soit", "pour", "et", "dans", "sont"},
	"de": {"der", "die", "das", "und", "ist", "eine", "ein", "sei", "für", "mit", "dass", "von", "zu", "auf"},
	"vi": {"của", "và", "là", "các", "cho", "với", "một", "có", "số", "tìm", "chứng", "minh", "rằng", "trong"},
	// Croatian and Dutch are only told apart, so that a text in them that a row marks as English is not taken (spec 06.8).
	"hr": {"je", "i", "u", "na", "se", "su", "od", "da", "za", "koji", "koja", "koje", "neka", "ako", "broj", "sve", "odredi", "kojima", "zadatak"},
	"nl": {"het", "een", "van", "zijn", "op", "voor", "niet", "wat", "hoe", "bepaal", "er", "dan", "ook", "welke", "elke", "getal"},
}

// detectOnly are the languages that Lang tells apart but does not give: the content has no texts in them, so a text in one of them is dropped.
var detectOnly = []string{"hr", "nl"}

// Lang gives the language of a text, from the language that the row names and from the text. A row may name two languages, as "English; Russian", or none. The text decides between them: its script, then its common words. It gives "" when the language is not clear.
func Lang(text, named string) string {
	var want []string
	for _, n := range strings.FieldsFunc(strings.ToLower(named), func(r rune) bool { return r == ';' || r == ',' }) {
		if c := languages[strings.TrimSpace(n)]; c != "" {
			want = append(want, c)
		}
	}
	var latin, cyrillic, han, kana, hangul, arabic int
	for _, r := range stripMath(text) {
		switch {
		case unicode.Is(unicode.Latin, r):
			latin++
		case unicode.Is(unicode.Cyrillic, r):
			cyrillic++
		case unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			kana++
		case unicode.Is(unicode.Han, r):
			han++
		case unicode.Is(unicode.Hangul, r):
			hangul++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		}
	}
	got := ""
	switch {
	case kana > 0:
		got = "ja"
	case hangul > latin/4 && hangul > 0:
		got = "ko"
	case han > latin/4 && han > 0:
		got = "zh"
	case arabic > latin/4 && arabic > 0:
		got = "ar"
	case cyrillic > latin/4 && cyrillic > 0:
		// The Cyrillic languages are not told apart by their script, so the row must name one.
		for _, w := range want {
			if w == "ru" || w == "mk" || w == "mn" {
				return w
			}
		}
		return ""
	default:
		var counts map[string]int
		got, counts = byWords(text)
		if got == "" && len(want) == 1 && latinLang(want[0]) && !otherWords(counts, want[0]) {
			// A text of mostly formulas has few words, so the language that the row names stands.
			return want[0]
		}
	}
	if got == "" || contains(detectOnly, got) {
		return ""
	}
	if len(want) == 0 || contains(want, got) || got == "en" {
		return got
	}
	return ""
}

// latinLang tells if a language is written in the Latin script.
func latinLang(c string) bool {
	_, ok := stopWords[c]
	return ok && !contains(detectOnly, c) || c == "af"
}

// otherWords tells if the common words of another language outnumber those of the named language, as in a Spanish text that a row marks as English. The Romance languages share many common words, so byWords may not tell which of them it is.
func otherWords(counts map[string]int, named string) bool {
	for lang, c := range counts {
		if lang != named && c >= 3 && c > 2*counts[named] {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// byWords gives the Latin language whose common words are the most frequent in the text, when they are clearly more frequent than the words of the next language, and the count of the common words of each language.
func byWords(text string) (string, map[string]int) {
	counts := map[string]int{}
	words := strings.FieldsFunc(strings.ToLower(stripMath(text)), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, w := range words {
		for lang, list := range stopWords {
			if contains(list, w) {
				counts[lang]++
			}
		}
	}
	best, next := "", 0
	for _, lang := range []string{"en", "es", "pt", "fr", "de", "vi", "hr", "nl"} {
		switch c := counts[lang]; {
		case best == "" || c > counts[best]:
			next = counts[best]
			best = lang
		case c > next:
			next = c
		}
	}
	if counts[best] < 1 || counts[best] < 2*next {
		return "", counts
	}
	return best, counts
}

// mathRE finds the formulas of a text.
var (
	// detailsRE is a folded block of a post, such as the original wording or a note of the poster.
	detailsRE = regexp.MustCompile(`(?is)<details>.*?</details>`)
	// hideNoteRE is the note at the end of the posts of some AoPS collections.
	hideNoteRE = regexp.MustCompile(`(?i)PS\.? You (?:should|can) use hide for answers\.?\s*(?:Collected \[here\]\([^)]*\)\.?)?`)
	// tagRE is an HTML tag of style, which the text does not need.
	tagRE = regexp.MustCompile(`(?i)</?(?:span|u|b|i|em|strong|font|center|sup|sub)(?:\s[^<>]*)?>`)
	brRE  = regexp.MustCompile(`(?i)<br\s*/?>`)
	// bbRE is a BBCode tag of style of AoPS.
	bbRE   = regexp.MustCompile(`(?i)\[/?(?:color|size|font|center|u)(?:=[^\]]*)?\]`)
	bbList = strings.NewReplacer("[list]", "\n", "[list=1]", "\n", "[list=a]", "\n", "[/list]", "\n", "[*]", "\n- ", "[i]", "*", "[/i]", "*", "[b]", "**", "[/b]", "**")
	// linkRE is a web link. A text with a link sends the reader to a page for a part of the problem or for its context.
	linkRE = regexp.MustCompile(`https?://|www\.[a-z]`)
)

// markup removes the HTML and the BBCode of a post that only style the text, the folded blocks and the notes of the poster.
func markup(text string) string {
	text = detailsRE.ReplaceAllString(text, "")
	text = hideNoteRE.ReplaceAllString(text, "")
	text = brRE.ReplaceAllString(text, "\n")
	text = tagRE.ReplaceAllString(text, "")
	text = bbRE.ReplaceAllString(text, "")
	return strings.TrimSpace(bbList.Replace(text))
}

var mathRE = regexp.MustCompile(`(?s)\$\$.*?\$\$|\$[^$]*\$`)

func stripMath(text string) string { return mathRE.ReplaceAllString(text, " ") }

// stepWords gives the number of distinct words of the steps of a solution in a text.
func stepWords(text string) int {
	seen := map[string]bool{}
	for _, w := range stepWordRE.FindAllString(text, -1) {
		seen[strings.ToLower(w)] = true
	}
	return len(seen)
}

// cutReply cuts a message of the translator, and all that follows it, from the end of a text. The message is in a paragraph after the first one; a text that starts with such a message is left as it is.
func cutReply(text string) string {
	m := replyRE.FindStringIndex(text)
	if m == nil {
		return text
	}
	if k := strings.LastIndex(text[:m[0]], "\n\n"); k > 0 {
		return strings.TrimSpace(text[:k])
	}
	return text
}

// firstCopy gives the first copy of a text that a dataset gave twice, in most cases a translation and the text again in other words: "Solve the system of inequalities: $$...$$\n\nSolve the following system of inequalities: $$...$$". The two copies are split at a blank line and must be close in length and in their word 5-grams, and have the same numbers: the parts of a problem, as "a) for $n=3$" and "b) for $n=4$", have other numbers.
func firstCopy(text string) string {
	paras := strings.Split(text, "\n\n")
	for k := 1; k < len(paras); k++ {
		a, b := strings.Join(paras[:k], "\n\n"), strings.Join(paras[k:], "\n\n")
		na, nb := utf8.RuneCountInString(a), utf8.RuneCountInString(b)
		if na < canon.MinText || nb < canon.MinText || 4*na < 3*nb || 4*nb < 3*na {
			continue
		}
		fa, fb := matchText(a), matchText(b)
		if shapeOf(fa).numbers == shapeOf(fb).numbers && match.Jaccard(match.Shingles(fa), match.Shingles(fb)) >= Twice {
			return a
		}
	}
	return text
}
