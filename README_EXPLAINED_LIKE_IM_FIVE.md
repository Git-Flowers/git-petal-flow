# GitHero-flow, explained like you're five

## The map

Imagine your project is a big map of roads. Every road is a **branch**. A
branch is a road where you can build something new without wrecking the road
next to it.

- **`main`** is the road that real people drive on. It must always work.
- **`develop`** is the road where everyone's new things get put together.
- A **feature** is a little side road you build to add one new thing.

## Saving your work

Every time you finish a piece of building, you take a **photo** of the road.
Git calls the photo a **commit**. Photos never disappear, so you can always look
back at how the road looked before.

## Joining roads

When your side road is done, you connect it back to the big road. Git calls
that a **merge**.

Here is the important rule in this project:

> Don't erase and redraw the map. Just add new roads.

So GitHero-flow always joins roads with a **merge commit**. It leaves a visible
mark that says "this side road joined here". It never moves the road to look
like it was always there, and it never rebuilds the road from scratch. The
map stays honest.

## Epics: a big road with small roads in it

Sometimes a lot of little side roads belong together, like all the roads that
make up one new town. That group is an **epic**.

- An epic has a number, like `epic/001-auth`. The number is its name tag.
- A **child** is a side road that belongs to the epic. Its name says so:
  `feature/epic-001/login-form`. The `001` says "I belong to epic 001".
- A child joins the epic first. When every child has joined, the whole epic
  joins `develop`.

The rule is: **the epic can't join until all its children have joined.** If
one child is still a half-built road, GitHero-flow says "not yet, finish that one
first" and names it.

## Releases: the flag on the hill

When the town is ready for people, you plant a flag on it. A **release** is
just a flag. In GitHero-flow it is a **tag** with a version number, like `v1.4.0`.

The flag does not hold any extra roads. It just says "this exact map is the
one we shipped."

## Why GitHero-flow is here

Git is great at remembering every photo. It is not great at remembering **which
road was which** and **who belongs to whom**. After a while you might ask:

- "Which feature was this part of the epic?"
- "Did I finish the login road or not?"
- "Did we already put version 1.4.0 on the hill?"

GitHero-flow writes those answers down for you. It keeps a small notebook beside the
map, in the git folder, with:

- which roads were made, and which road each one came from,
- which roads are finished,
- which photos (commits) went onto which road.

It also does the joining the same safe way every time, and it warns you before
you make a mistake. It never erases a road, never rebuilds one, and never
pushes anything for you. You still decide when to share the map.

## The whole story in one picture

```text
main ─────────────────────────────────── flag v1.4.0 ──────────►  (the shipped map)
  │
develop ──●────────────────────────────────────────●──────────────►
            \                                     /
             epic/001-auth ──●───────────●──────●   (joins develop when all children joined)
                              \          /
   feature/epic-001/login ─────●──────── joins the epic
   fix/epic-001/session  ──────────●──── joins the epic
```

Read it like a story:

1. Build a town (the epic) with a number on its sign.
2. Build the small roads inside it (the children), each with the town's number.
3. Join each small road to the town.
4. When the town is complete, join it to the big road.
5. When it's time to ship, plant a flag with a version number.

That's it. Add new roads, never erase the old ones, and always leave a note of
where you joined.
