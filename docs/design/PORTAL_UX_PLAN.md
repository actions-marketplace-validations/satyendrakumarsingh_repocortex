# Portal UX plan

A design for the next RepoCortex portal, worked out from who uses it and what
they come to do. It replaces the current portal's structure; it is not a restyle
of it. Written 2026-11-12.

**Status.** The first cut is built and runs on live data:

```bash
repocortex ui --next
```

It serves Home, search, the symbol page with five tabs, a page each for a
file, a route and a table, Explore, Change (for one symbol), Health and Agents.
The current portal (`repocortex ui`) is unchanged and stays the default until
this one covers every screen. A static prototype of the
whole design, including the pages not yet live, is in
[`portal-prototype/index.html`](portal-prototype/index.html); open the file in a
browser.

## Contents

- [The problem with the current portal](#the-problem-with-the-current-portal)
- [Who uses it, and for what](#who-uses-it-and-for-what)
- [Design principles](#design-principles)
- [Structure](#structure)
- [The screens](#the-screens)
- [Components](#components)
- [Visual language](#visual-language)
- [States](#states)
- [Accessibility and responsiveness](#accessibility-and-responsiveness)
- [Engineering approach](#engineering-approach)
- [Delivery phases](#delivery-phases)
- [How we will know it worked](#how-we-will-know-it-worked)
- [Audit of the first cut](#audit-of-the-first-cut)
- [Audit of the file, route and table pages](#audit-of-the-file-route-and-table-pages)
- [What the tools cannot feed yet](#what-the-tools-cannot-feed-yet)
- [Open questions](#open-questions)

## The problem with the current portal

The current portal is 23 screens in one flat list. Each screen is a report on
one feature of the index. Reviewing it against what people come to do:

| What is there today | Why it costs the reader |
|---|---|
| 23 destinations in the rail | The reader must know which feature answers their question before they can ask it |
| The same symbol typed into each screen | Callers, blast radius, tests, owners and the refactor plan are five screens, each with its own input; nothing carries the symbol across |
| A home page that is an inventory | Counts of files and relationships come before any answer; three promotional cards come before those |
| Search is one screen among 23 | The most common starting point is not reachable from where the reader lands |
| Unsupported claims in the chrome | A "Verified Index Guarantee" badge is shown on every run, including degraded ones |
| Emoji as icons, arrows on buttons, upper-case labels | Decoration that carries no information and makes every panel equally loud |
| No view of what an agent did | The portal describes the index; it cannot show what an agent asked of it |

The individual screens are careful about evidence and confidence, and that is
kept. The structure around them is what changes.

## Who uses it, and for what

Four people, from the README's own "who it's for", with the question each
arrives with.

| Person | Arrives asking | Today that takes |
|---|---|---|
| A developer about to change something, often with an agent | "What calls this, what breaks if I change it, and what do I run?" | Four screens and four inputs |
| A reviewer or tech lead | "What does this pull request put at risk, and who should look at it?" | The command line; the portal has no change-level view |
| An engineer new to the codebase | "How is this laid out, and where does a request go?" | Picking from six "how it runs" screens |
| The person who approved the tool | "Can I trust this index, and what has the agent been reading?" | Two screens for trust; nothing for the agent |

The first is by far the most frequent. The design is built around that person
and must not get worse for the other three.

## Design principles

1. **Start from the question.** The reader's first act is to name the thing
   they care about. A search field is on every screen, and the home page is one.
2. **One page per thing.** A function, a file, a route or a table has a page.
   Everything the index knows about it is on that page, as tabs. The reader
   names the thing once.
3. **Answer in a sentence, then show the evidence.** Every page opens with one
   sentence a person could say aloud: "Called from 11 places. A change can reach
   70 symbols, and 7 tests reach it." The tables follow.
4. **Say what the answer cannot see, beside the answer.** The limits are in a
   fixed place on every page, not in a footer and not on a separate screen.
5. **One visual grammar for confidence.** Four tiers, four fills, the same
   everywhere: in the legend, beside every row, and on the map.
6. **Every answer is also a command.** Beside each answer is the tool call or
   command that returns it, so a person can hand the question to their agent.
7. **Nothing claims more than the index knows.** No badge, score or adjective
   that a degraded run would also receive.

## Structure

Five sections replace twenty-three screens. Each is named for what the reader is
doing.

```text
+-------------------------------------------------------------------------+
| repocortex 1f68eb5   [ Search a symbol, file, route or table       / ]  |  82% bound | 0 proven
+------------+------------------------------------------------------------+
| Home       |                                                            |
| Explore    |   page                                                     |
| Change     |                                                            |
| Health     |                                                            |
| Agents     |                                                            |
+------------+------------------------------------------------------------+
```

| Section | The reader is | Holds (current screens it absorbs) |
|---|---|---|
| **Home** | Asking a question | Search; what needs attention; the confidence legend; tasks. (Index health) |
| **Explore** | Learning the layout | Map; routes and what they touch; tables; events; settings; across repositories; shape of the code |
| **Change** | Checking a change | A branch, commit range or diff: what it reaches, tests to run, coupled files, who to ask, the plan. (What could break, plan a refactor, task brief, check a change, what changed, who knows this code) |
| **Health** | Looking for what to fix | Hotspots; unused code; blind spots; architecture rules; dependencies; test coverage; accuracy |
| **Agents** | Connecting an agent or checking on one | Connect; tool profiles and their cost; recent calls; retrieval quality |

And one kind of page that belongs to no section, reached from search and from
every name anywhere in the portal:

| Page | Tabs as designed | Tabs as built |
|---|---|---|
| **Symbol** | Callers, Calls, What could break, Tests, People | Callers, Calls, What could break, Tests, Type hierarchy |
| **File** | Definitions, Depended on by, Changes with, History | Start of the file, Changes with, Looks unused, Routes |
| **Route** | Handler, Tables touched, Callers in other repositories | What it calls, Tables it reaches, Tests |
| **Table** | Readers, Writers, Routes that reach it | Written by, Read by, Routes that reach it, Declared in |

On the file, route and table pages, where the built tabs differ from the
designed ones the tool surface is the reason, and
[What the tools cannot feed yet](#what-the-tools-cannot-feed-yet) names what
each missing tab is waiting for.

The top bar is constant: the repository and the commit it was indexed at,
search, and a trust indicator (share of references bound, count compiler-proven)
that opens the accuracy view.

Every name on every page is a link to that thing's page. This is the change that
removes the re-typing.

## The screens

### Home

```text
What do you want to know about repocortex?
Search for a function, type, file, route or table. Its page shows what calls it...
[ Try it with openReadOnly ]  [ Review a change ]

Needs your attention ------------------------------------------------------
  Nothing in this index is compiler-proven yet        (one command fixes it)
  2 widely used functions have no test the index can find
  internal/web/web.go is where change concentrates

How far can you trust an answer here? -------------------------------------
  [=========== very likely ===========|//|:::: a guess ::::]
       0  proven by a compiler   Safe to act on without checking.
  15,513  very likely            Act on it; it is rarely wrong.
     219  uncertain              Check it before relying on it.
   3,575  a guess                Hidden unless you ask for it.

Start from a task ---------------------------------------------------------
  See who calls something        Find your way around      Connect your agent
  Check what a change breaks     Find what to clean up     See what it asked
```

"Needs your attention" is at most three items, each a finding with its remedy,
ordered by how much acting on it improves every other answer. When there is
nothing to report it says so in one line.

### Symbol page

```text
internal/cli/opendb.go:43
openReadOnly   [function] [unexported]

Called from 11 places. A change can reach 70 symbols, and 7 tests reach it.
Risk is moderate.

 Callers 11 | Calls 3 | What could break 70 | Tests 7 | People
-----------------------------------------------   +------------------------+
 Called from        How sure        Evidence      | What this list cannot  |
 agentToolNames     [//] very likely  agentsetup.go:306 | see                    |
 verifyGraph        [//] very likely  commands.go:948   | Ask your agent the     |
 ...                                                    | same thing             |
 Showing 8 of 11. Show all                              | Before you change it   |
                                                        +------------------------+
```

The sentence at the top is the answer. The tabs are the evidence for each clause
of it, with the count on the tab so the reader sees the size before clicking.
The column at the right is always the same three things: what this view cannot
see, the tool call that returns it, and the next step.

### Change

The reviewer's page, and the one the current portal has no equivalent of.

```text
What does this change put at risk?
Comparing main with feat/http-transport.   [ change ]

It changes 3 functions in 1 file. Those can reach 18 more symbols. Risk is low.

  3 functions changed   18 within reach   4 tests to run first   1 coupled file

What changed, and what each can reach      | Changes with it, no code link
Run these tests first  (with the command)  | Who to ask
                                           | Put this on the pull request
```

"Changes with it, no code link" is the hotspot coupling data placed where it is
useful: at the moment someone is about to miss a file the call graph cannot
connect.

### File page

Someone opens a file's page when they are about to edit the file, or have been
sent a diff that touches it. They are asking: how much is in here, how much
leans on it, does it move with anything I would not think to check, and is any
of it dead weight?

```text
internal/cli/
install.go   [file]

Holds 20 definitions. Code in other files uses them in 13 places.
Changed in 8 of the last 47 commits, most recently on 2026-11-12.

 Start of the file | Changes with 1 | Looks unused 1 | Routes 0
-----------------------------------------------   +------------------------+
 File            Changed together in              | What this answer       |
 .env.example    3 of the 8 commits that          | cannot see             |
                 changed this file                | Ask your agent the     |
 The whole repository was searched.               | same thing             |
                                                  | Before you change      |
                                                  | this file              |
                                                  +------------------------+
```

The sentence is three figures from one answer (`list_hotspots` asked about this
file alone): how much it holds, how many uses in other files point into it, and
how often it changes. The tabs:

| Tab | The reader is asking | Fed by |
|---|---|---|
| Start of the file | What does this file say it is, and what does it bring in? | `get_file_outline`. Every word in it that `find_symbol` confirms is defined in this file becomes a link to that definition's page |
| Changes with | What else do I have to open when I change this? | `list_hotspots`, asked of the whole repository and then of each folder on the way down to the file |
| Looks unused | Is any of this dead? | `list_dead_code` for this path, at the weaker certainty and with exported names, because the reader is looking at one file and can check each |
| Routes | Does a request arrive here? | `list_api_endpoints`, kept to the routes declared here or answered by a function here |

"Changes with" is asked more than once on purpose. The history tool names
partners only for the files it ranks, and only partners inside the folder it was
asked about, so asked about one file it can name none. The page asks the whole
repository first and narrows a folder at a time until the file is ranked, then
says which search it is showing: a search of one folder cannot see a partner
outside it, and a list that looked complete would be the wrong thing to show.

The third margin note is the page's own limit, stated as a next step: what uses
a definition, which tests reach it and who last changed it are answers about one
definition, so the page sends the reader to the definition.

### Route page

Someone opens a route's page when an endpoint is misbehaving, or they are about
to change what it does. They are asking: which function answers this, what does
that function go on to touch, and what would tell me if I broke it?

```text
Declared at internal/web/web.go:359
GET /api/db   [route]

Answered by handleDB. It reaches 1 table. 8 tests may cover it; none is tied
to it by a call.

 What it calls 15 | Tables it reaches 1 | Tests 8
---------------------------------------------------------------------------
 Table            Nearest use   How sure         By way of
 table_accesses   reads it      [//] very likely handleDB > TableUsageReport
                                                  > table_accesses
```

The sentence is said as its parts arrive: who answers the route at once, then
what that function reaches, then whether a test covers it. "May cover it; none
is tied to it by a call" is the difference between a test found through the
call graph and one found by its name or its place in the tree, said in the
sentence because it changes what the reader should do next.

| Tab | The reader is asking | Fed by |
|---|---|---|
| What it calls | What happens first when this is called? | `find_callees` on the function that answers the route |
| Tables it reaches | What data does a request to this end up touching? | `list_db_tables`, then `trace_call_path` from the function to each table |
| Tests | What do I run? | `find_tests` on the function |

"Tables it reaches" is one question to the index for each table, so the panel
counts them off as they are answered and the page asks about the forty most
used tables unless the reader asks for all of them. Each row shows the chain of
calls that reaches the table, each function in it a link, and the weakest link's
confidence, because a chain is as sure as its least sure step.

### Table page

Someone opens a table's page before changing a column, or when a row has gone
wrong and they need to know what could have written it. They are asking: what
writes to this, what reads it, and which requests end up here?

```text
Database table
table_accesses   [table]

7 statements read it and 3 write to it. It is declared in this repository.
3 routes lead to it.

 Written by 3 | Read by 7 | Routes that reach it 3 | Declared in 2
```

Writers come first. What writes a table is the list to check before changing a
column, and mixing it with readers is worse than not answering.

| Tab | The reader is asking | Fed by |
|---|---|---|
| Written by | What could have put this row here? | `find_references` on the table, the rows that write |
| Read by | What breaks if I rename this column? | The same answer, the rows that read |
| Routes that reach it | Which requests end up here? | `list_api_endpoints`, then `trace_call_path` from each route's function to the table |
| Declared in | Where is its shape defined? | `find_symbol` for tables of this name; indexes and views built on it are listed beneath |

A table that nothing in the repository declares still has a page. The index has
a count of the statements that name it and no list of the code behind them, and
the page says that and offers a text search, which is the honest next step.

### How the three pages connect

Nothing is a dead end. A place (`file:line`) anywhere in the portal is a link to
that file's page. A route's name is a link to the route's page, a table's to the
table's, a function's to the symbol's. Search finds a file by its whole path, a
route as it is written (`GET /users`) and a table by its name. The symbol page
of a function that answers a route says so and links to the route; the symbol
page of a table or a route links to the page made for it. Explore lists every
route and table as links, and Health and Change link every file they name.

### Health

Three lists in order of usefulness: hotspots (score, commits, size, depended on
by, changes with), unused code (the likely tier only, with the count of what is
merely possible and why), and what the index could not see, each with its
remedy. Architecture rules, dependencies and accuracy are further tabs.

### Agents

Four ways to connect, each with its one command. A table of the three tool
profiles with the context each costs per request, so the choice is made with the
price visible. Then recent calls from the audit log: time, surface, tool, what
was asked, how long it took, how large the answer was. Selecting a call replays
it, which is how a person sees exactly what their agent was shown.

Recent calls are designed and not built. The audit log is a file on the
machine, and no tool reads it back, so the page has nothing to draw them from
and does not invent a route of its own to get them. In their place it says so,
and gives the command that turns the record on and the one that reads the file.

## Components

A small set, each used the same way everywhere.

| Component | Purpose | Rule |
|---|---|---|
| Verdict sentence | The answer, in words | One per page, directly under the title. No more than two sentences |
| Tier mark | How a relationship was established | A patterned swatch with its word. Never colour alone |
| Evidence table | The rows behind a verdict | Name (a link), how sure, file and line. Sorted by confidence, then location |
| Row list | A set of things to go to or act on | Title as the link, one line of explanation. No card, no icon |
| Margin note | What an answer cannot see; the command; the next step | A ruled left edge. Amber edge only for a limit the reader should check |
| Command block | The tool call or command behind an answer | Monospace, copyable, never truncated |
| Fact row | A handful of counts that frame a page | Large figures in the display face. At most four |
| Tabs with counts | Facets of one thing | The count is on the tab. Zero is shown as zero |

Not in the set, deliberately: cards, icons, badges that state a quality, charts
drawn for decoration, and any second way of showing confidence.

## Visual language

Called Survey: a map of a codebase that looks like a map.

| Choice | Value | Why |
|---|---|---|
| Light theme | Ground `#f4f6f9`, white sheets, navy ink `#0e1b2c`, accent petrol teal `#0b7285` | Body text 15.9:1 on a sheet; the quietest text 4.9:1 |
| Dark theme | Ground `#0b1118`, sheets `#121a24`, ink `#e9eff7`, accent cyan teal `#4fc6d8` | Chosen on its own terms, not inverted from light. Body text 14.6:1; the quietest text 5.6:1 |
| Separation | A hairline, never a shadow | Flat sheets and ruled lines |
| One accent | Links, the current section and the primary action | Nothing else is coloured |
| Status | Amber for a limit to check, green for a healthy state, red for a failure | Never the only signal; always beside a word |
| Headings and figures | The DIN family from the reader's system (DIN Alternate, Bahnschrift) | The lettering of technical drawings. No font is downloaded |
| Running text | The system interface face | Legible at 15px, already on the machine |
| Monospace | For code only: paths, symbols, commands | Not for labels |
| Confidence | Four fills: solid, close hatch, open hatch, dots | Reads in greyscale and under any colour blindness |
| Shape | 5 to 8px radius on controls, none on rules | Flat sheets and ruled lines |
| Labels | Sentence case | Upper case makes a label louder than the value it names |

The one loud element is the confidence legend on the home page. Everything else
is quiet so that it can be.

Both themes are held to WCAG AA by a test (`TestThemesMeetContrast` in
`internal/portal`), which checks twenty-one foreground and background pairs in
each. It caught a control border at 2.92:1 in the first light palette.

## States

| State | What the page shows |
|---|---|
| No index | One sentence and the command that builds one. No empty tables |
| Index being built | Progress, and that answers will appear when it commits |
| Nothing found | "No callers found", then what that does and does not mean, then what to try |
| Low coverage | The warning replaces the verdict sentence. A number computed from a quarter of the evidence is not shown as an answer |
| Truncated | "Showing 50 of 340", what was left out, and how to narrow it |
| Index older than HEAD | A line under the top bar on every page, with the command to refresh |
| Failure | What failed, in the interface's voice, and what to do. No stack trace, no apology |

## Accessibility and responsiveness

- WCAG 2.2 AA. Contrast is solved for both themes, not eyeballed: 4.5:1 for
  text, 3:1 for control edges and focus rings.
- Every control is reachable by keyboard, in reading order. `/` focuses search.
  Focus is always visible.
- Confidence is never colour alone: pattern and a word.
- Tabs, tables and the legend carry the correct roles and labels. The legend bar
  has a text equivalent.
- Motion only in response to an action, and none under `prefers-reduced-motion`.
- Below 900px the navigation becomes one scrolling row, the margin notes move
  under the content, and tables scroll inside their own container. The page
  never scrolls sideways.
- Light and dark follow the system and can be set by hand.

## Engineering approach

The constraints the current portal already meets are kept.

| Constraint | Consequence |
|---|---|
| Served from the binary, no build step, no network | Plain ES modules and one stylesheet, embedded. No framework, no bundler, no CDN |
| Content Security Policy forbids other origins | System fonts only |
| Read-only | `GET` routes only, as today |
| One tool surface | The portal reads the same REST endpoint agents use (`/api/v1/tools/{name}`) for every per-symbol view, so a page and an agent cannot disagree |

That last row is the structural change underneath. Today the portal has its own
JSON routes beside the agent tools. Built on the tool endpoint, the symbol page
is five tool calls rendered as tabs, and "ask your agent the same thing" is
literally true.

Shape of the code:

```text
assets/
  index.html          the shell: top bar, navigation, one <main>
  app.css             tokens, then components, in that order
  js/router.mjs       hash routes to views; back and forward work
  js/api.mjs          one fetch wrapper over /api/v1
  js/components/      verdict, tier mark, evidence table, margin note, tabs
  js/views/           home, symbol, file, route, table, change, health, agents, explore
```

Each view is a function from data to DOM. No view is longer than a screen of
code; the current 11,800-line script is not carried over.

## Delivery phases

Each phase ships something usable and leaves the current portal working until
the last.

| Phase | Ships | Status |
|---|---|---|
| 1. Shell and symbol page | Top bar with search, five-section navigation, the symbol page with all five tabs, served at `repocortex ui --next` | **Built** |
| 2. Home and Change | Home with the index's state and the legend; the change page | **Built** for one symbol. A commit range needs the `analyze_change` tool |
| 3. Health and Agents | Coverage, hotspots, unused code, blind spots; connect, what each tool costs | **Built**. Recent calls with replay need the audit log exposed as a tool |
| 4. Explore and the remaining pages | Map, routes, tables, events; file, route and table pages | File, route and table pages **built**, and Explore lists routes and tables as links to them. The map, events and settings are not built: no tool lists events or settings |
| 5. Switch over | `repocortex ui` serves the new portal; the old one is removed; end-to-end tests and the portal guide are rewritten | Nothing links to the old portal |

Phase 1 is the test of the whole idea. If one page per thing does not beat five
screens for the first person in the table, the later phases change.

## How we will know it worked

Measured with five people per round, on their own repository, before and after.

| Measure | Now (estimated from the flow) | Target |
|---|---|---|
| Screens visited to answer "what calls this and what breaks" | 3 to 4 | 1 |
| Times the symbol name is typed for that question | 3 to 4 | 1 |
| Time to the first correct answer for a new user | Not measured | Under 30 seconds |
| People who notice an answer's limits unprompted | Not measured | 4 of 5 |
| People who can say what "very likely" lets them do | Not measured | 4 of 5 |

The first two rows are counts of the design and are met by construction. The
last three need the sessions.

## Audit of the first cut

The first cut was used as a reader would use it, on this repository's own index,
and each thing that got in the way was fixed before it was called done.

| What a reader hit | Fix |
|---|---|
| The home page showed three sentences written for a model ("resolved 19076 of 23299 references that name something this index defines") | Each is reduced to its one figure in plain words, with the full sentence one click away |
| The home page showed nothing until its slowest check finished | Each line is asked for and drawn on its own |
| Tabs gave no idea how much was behind them | Every tab's answer is fetched at once and its count shown on the tab |
| No way to hide the weak relationships | A two-way filter on each relationship tab: everything, or very likely and better |
| A tab change lost its place on reload and in the back button | The tab and the filter are in the address |
| Coverage was one long sentence, repeated three times | Drawn as a percentage and three counts, with the command that improves it |
| Commands had to be selected by hand | Every command has a copy button |
| A column was headed with the resolver's own word | "How it was found" |
| The page title capitalised a function's name | A symbol's name is shown as written |
| The theme button's label went stale when the system theme changed | It follows the system until the reader chooses |
| A light-theme control border was below 3:1 | Darkened; pinned by the contrast test |
| The tier words differed from the tool's | The page says proven, very likely, uncertain and a guess, and says what each lets you do |

Still to do, found in the same pass: the Change page takes one symbol and not a
branch; recent agent calls are not shown; and the type-hierarchy tab is offered
for functions, where it is always empty. The pages for a file, a route and a
table have since been built, and are audited below.

## Audit of the file, route and table pages

The three pages were walked on this repository's own index, in both themes, at
desktop and phone width, and with the keyboard alone.

| What a reader hit | Fix |
|---|---|
| "Changes with" said nothing changes with any file. The history tool names partners only inside the folder it is asked about, so asked about one file it named none, and the page reported that as a finding | The question is asked of the whole repository first and then a folder at a time; the page says which search it is showing and what that search cannot see |
| A file's outline stops at twenty lines, which for a file with a long opening comment is the comment | The page says how many lines it has, that the rest is not listed, and gives the command that prints all of it. Names in those lines that are defined in the file are links |
| A tool that failed drew as "Nothing found" | A failure is drawn as a failure, with what went wrong and that nothing can be concluded from it |
| At phone width a wide table or block of code made the whole page scroll sideways | The column that holds it may shrink, so the table scrolls inside its own frame |
| On a phone the search field was squeezed to a few letters by the repository's name | Search has a row to itself below 900px |
| In a narrow table a path or a name broke mid-word into a column three letters wide | Names and places keep their words whole; the table scrolls instead |
| A file's page listed its own path on every row of "Looks unused" | The column shows the line |
| The note beside a file's history explained how to read a ranking of files | Left out: the page asks about one file |
| The function's name inside the route's sentence was a third the size of the sentence | A name in a sentence is set at the sentence's size |
| A route's name wrapped between its method and its path | A name in a table stays on one line |
| "Holds 1 definition. Code in other files uses them" | The sentence agrees in number |
| A call the index could not connect was a link to a symbol page that said "not found" | It is text |

## What the tools cannot feed yet

The portal has no data routes of its own, so a part of a page exists only where
a tool answers its question. These parts are designed and not built, and each is
waiting for the one thing named beside it.

| Not built | Why | What it needs |
|---|---|---|
| File page: every definition in the file | `get_file_outline` is cut at twenty lines by the bound every answer shares, and no tool lists a file's definitions | A tool that lists the definitions in a file (name, kind, line), or the outline exempted from the excerpt bound |
| File page: what depends on it, as a list | `list_hotspots` gives the count of uses from other files and not the uses. Callers are known per definition, and the definitions cannot be listed | The same tool: with a file's definitions, the page can ask `find_callers` for each |
| File page: tests that reach it, and who knows it | `find_tests` and `find_owners` take one symbol and no path | A `path` argument on each, or the list of definitions to ask them about |
| Route page: settings it reads and events it sends | `get_config_key` looks up one key by name, and no tool lists keys or events | Tools that list configuration keys and events, as `list_db_tables` lists tables; `trace_call_path` can then follow a route to each |
| Route page: callers in other repositories | The system graph is a command (`repocortex system`), not a tool | A tool over the federated graph |
| Table page: every reader and writer of a busy table | `find_references` returns fifty rows, shared between readers and writers | A filter on the relationship (reads or writes), or paging |
| Agents: recent calls | The audit log is a file no tool reads | A tool that returns the latest lines of the audit log the server is writing (time, how the call arrived, tool, arguments, how long, how large, outcome) |

## Open questions

1. **Does Explore need the force-directed map on its first screen?** It is the
   most distinctive view and the least often useful one. The plan puts it one
   click in; a session with new engineers should settle it.
2. **Should recent calls be on by default?** They need the audit log, which is
   opt-in. The Agents page can offer to turn it on, but must not do so itself.
3. **Where does a task brief live?** It is placed under Change. It could instead
   be the entry point of the Agents section, since an agent is its main reader.
4. **Is five sections right for a repository with no routes or tables?** Explore
   thins out. It may need to adapt to what the index holds.
