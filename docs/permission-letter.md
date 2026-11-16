# Permission letter

This is the template of the letter that asks a rights holder for leave to show its competition texts in full on olymmath (spec 11.4). Each row of `registers/permissions.yaml` in the content repository gets one letter made from it. The owner of olymmath reads and sends each letter; agents prepare rows and letters and never send them.

## How a letter is made from a row

- Fill each `{placeholder}` from the row: `{holder}`, `{competitions}` (the names of the competitions of the row), `{years}`, `{texts}` (statements, official solutions, or both, from `scope`), and `{example}` (the olymmath page of one of the holder's problems, so the holder sees what is shown today).
- Keep option 1 always. Keep option 2 when `scope.uses` has `export`, and option 3 when it has `licence`.
- When the row has `languages` other than `en`, prepare a translation of the filled letter into that language and send both, the translation first. A translation is checked by a person who reads the language, or marked in the letter as a machine translation of the English text, which is the one that counts.
- Send it to the contact of the row, from the project address, not a personal one. Store the letter as sent, set `letter` to its `sha256:` hash, set `dates.sent` and the state `sent`.
- No answer after 8 weeks: one short reminder, then the state `no-answer` after 8 more weeks. A refusal is final for that scope: the row becomes `refused`, and the texts stay in their zone (or move to Z3 if the holder asks, spec 11.6).
- A grant is recorded in the row (`granted`, `dates.answered`, `answer`) and applied by its own change of `registers/sources.yaml` (spec 11.4.1).

## The letter (English)

> Subject: Permission to show {competitions} problems on olymmath, a free archive
>
> Dear {holder},
>
> I run olymmath (https://olymmath.tamnd.com), a free, non-commercial archive of mathematics competition problems. It lists each problem with its competition, year and number, its official source, and solutions written by contributors. The site has no advertising, no paywall and no account; its code and its catalogue data are open (Apache-2.0 and CC0).
>
> Today olymmath shows the {competitions} problems as short references: the statement with your name and a link to your official page, and no official solutions. You can see one here: {example}.
>
> I would like to ask for your permission to show the full text of the {texts} of {competitions}, {years}, on olymmath. On every page the text would carry:
>
> - your name as the source and rights holder, for example "© {holder}, used with permission";
> - a link back to your official page for the competition;
> - no change to the wording, except typesetting (formulas in LaTeX); translations are labelled as translations, and your original stays the reference.
>
> If you agree, please tell me which of these you allow:
>
> 1. the full text on the olymmath site, as above;
> 2. also the text in olymmath's free data downloads (with the same attribution), which others may use for research and teaching;
> 3. an open licence such as CC BY 4.0 on these texts, if you prefer to let everyone reuse them with credit.
>
> A yes to option 1 alone already helps a lot. You can also limit it to some years or to the statements only.
>
> If you ever want a text changed or removed, write to takedown@olymmath.tamnd.com and it is done within 7 days. If you do not want olymmath to show your problems at all, even as references, tell me and they become links only.
>
> Thank you for running {competitions} and for publishing the problems.
>
> Yours sincerely,
> {sender}, olymmath

## Notes

- The letter makes no claim about the law (fair use, thin protection of short texts); it asks.
- Keep it short. A holder reads the first paragraph and the question; everything else is detail.
- The removal promise is the one of spec 11.6 and must stay true: do not send a letter while the takedown address does not work.
