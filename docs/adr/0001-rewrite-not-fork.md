# 0001 — Rewrite the generator instead of maintaining a wire fork

Status: accepted

## Context

solder started from google/wire's code (Apache-2.0). wire is archived. solder
wants a different API (lifecycle, modules, groups) and to own its codebase,
including its copyright and license choice.

Renaming identifiers or restructuring wire's files does not change where the
code came from: it stays a derived work and must keep wire's license text and
copyright headers.

## Decision

- Write a new generator from scratch against solder's own API (ADR 0002). It
  is designed from solder's requirements, not by editing wire's code.
- wire's generator, marker API, docs, tutorial and project files are deleted
  in the same change that adds the new generator. Git history keeps them.
- A file keeps the "Copyright The Wire Authors" header for as long as it
  contains code from wire. The header is removed only by deleting that code,
  never by editing it around.
- Once no wire code remains, solder gets its own license (MIT or Apache-2.0,
  decided then) under its own copyright. The repository is never left
  unlicensed: unlicensed code cannot be adopted by companies.

## Consequences

- The new runtime (ADR 0003) has no wire ancestry and is written new.
- Ideas are not copyrighted: code generation for DI, a provider graph,
  topological ordering and golden-file tests are fair to reuse as ideas.
