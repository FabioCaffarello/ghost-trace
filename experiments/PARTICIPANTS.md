# What volunteers are told

This is the consent script for the human capture study, in a file of its
own so that **a change to what is collected produces a diff a person can
review**.

That is the point of the file existing, and it is the second half of an
audit finding. The first half was that the wording used to be false — it
claimed no key events were collected, which stopped being true when the
keystroke channel was added. The wording was corrected. But it lived
buried in a 300-line README, where a change to the SDK could alter what
is collected without touching anything a volunteer would ever be shown.

`disclosure_test.go` compares this file against the vocabulary the SDK
actually emits, in both directions. Adding a collected value without
telling anyone fails the build; describing something that is not
collected fails it too.

---

## The script

> **What this is.** You are helping test software that tries to tell
> automated browser activity apart from human activity, by how the
> interaction *behaves* rather than by who you are.
>
> **What is recorded is how you interact, never what you type.**
> Specifically:
>
> - **pointer** — where the mouse or trackpad moves, and when.
> - **key** — the *timing* of key presses and releases, and a coarse
>   class for each one. The classes are **alpha** (a letter), **digit**
>   (a number), **whitespace** (space, tab, enter), **nav** (arrows,
>   home, end and similar), **mod** (shift, control, alt, command) and
>   **other**. Which class, and when. **Never the key itself.**
> - **scroll** — that you scrolled, and whether it came from a real
>   gesture or from the page moving itself.
> - **focus** — when a field gained or lost focus.
> - **visibility** — when the page became visible or hidden, which is
>   what happens when you switch tabs.
> - **form** — that a field was filled by pasting, by autofill, by a
>   submit, or with no typing behind it at all.
>
> **What is never recorded.**
>
> - The content of any key you press. The six classes above are what is
>   kept, and a class preserves the rhythm of typing while making the
>   text itself unreconstructable.
> - The value of any form field. You can type anything you like into
>   them, including nonsense.
> - Canvas, WebGL, font or audio fingerprinting. None of it is
>   collected.
> - Your name, email or any identifier you did not choose.
>
> **Which field you were in is recorded as an opaque code**, not as the
> field's name. The code is a hash of the field's tag, type, name and
> id, so two visits to the same form produce the same code — that is
> what makes the measurement possible — and the code cannot be read back
> into a field name by anyone holding it.
>
> **Nothing is written to your browser.** No cookies, no local storage,
> no session storage, no indexed database. Closing the tab leaves
> nothing behind.
>
> **Your participant code is a pseudonym you chose to accept**, and you
> can discard it. It appears in the link you were sent and in the
> recorded rows, and it is the only thing linking one visit to another.
>
> **You can stop at any time**, and you do not have to say why.

---

## What is NOT settled, and is not claimed above

Named here rather than papered over, because a consent script that
answers a question it has not decided is worse than one that says the
question is open.

- **Retention.** No retention period has been decided or promised. The
  captured rows live in `experiments/results/` on the machine that ran
  the study.
- **Deletion on request.** There is no implemented mechanism for a
  participant to have their rows removed. Rows carry the participant
  code, so removal is *possible* by hand; nothing automates it and
  nothing has been promised.
- **Who holds the data.** A single maintainer, on one machine. There is
  no institutional review, no data controller, and no third party.

The audit that produced this file recommended an RFC for human-study
data governance covering exactly these three, and gates reopening
recruitment on it. **That RFC has not been written.** Until it is, the
script above should not be handed to anyone.

---

## Changing what is collected

The vocabulary lives in `services/collector/internal/ingest/vocabulary.go`
and in `sdk.js`, and the two are already compared in both directions.
This file is the third party to that comparison.

To add a collected value:

1. Add it to `vocabulary.go` and `sdk.js` — `vocabulary_test.go` fails
   until both agree.
2. Describe it in the script above — `disclosure_test.go` fails until
   it is there.
3. Ask whether the script is still one a person would consent to. That
   part no test can do.
