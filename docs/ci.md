# CI and release workflows

What runs in GitHub Actions, what each workflow publishes, and how agents propose workflow
changes they can't push themselves.

## Workflows

| Workflow | Trigger | What it does |
|---|---|---|
| `go.yml` (Test Go Compile) | push to `main`, PRs to `main` | Build + `go vet`; the `-short` suite under `-race`; the `database`/`controllers` suites on Postgres and MySQL; full suite with coverage (updates the coverage badge gist on `main`). **Must pass before merge.** |
| `codeql-analysis.yml` | push/PR to `main`, weekly | CodeQL static analysis for Go. |
| `docker-image-beta.yml` | hourly (`17 * * * *`), manual | Gate job decides, then builds the image for `linux/amd64`, `arm64`, `arm/v7` and `386` and pushes `beta` and `beta-<short sha>` to Docker Hub (`aunefyren/treningheten`) and GHCR. See [Beta images](#beta-images). The version placeholder becomes `beta-<short sha>`. |
| `docker-image.yml` | release published | The same multi-arch image, tagged with the release tag and `latest`. |
| `release.yaml` | release published | Cross-compiles the binaries (`linux` on `386`/`amd64`/`arm64`/`arm`, `windows` on `386`/`amd64`) with `wangyoucao577/go-release-action` and attaches them to the release. |
| `tag-branch-with-version.yml` | any tag pushed | Writes the tag into the `{{RELEASE_TAG}}` placeholder in `files/config.go` and force-moves the tag onto that commit. |

The version placeholder is explained in [architecture.md](architecture.md).

### Toolchain pins

- The `Dockerfile` builder image and the CI jobs (`1.26.x`) follow the latest Go patch on
  their own.
- `release.yaml` pins an exact patch in `goversion`, because the action builds the download
  URL from it verbatim and a floating `1.26` doesn't resolve. Bump it on each Go patch
  release (see [security.md](security.md), S12).

## Beta images

`docker-image-beta.yml` does not run on every merge (a build takes 20-27 minutes under QEMU
and overlapping runs raced each other). It runs every hour at :17 and on manual dispatch,
never two at once (`concurrency: beta-image`, no cancelling).

A cheap **gate** job compares `main`'s HEAD with the `org.opencontainers.image.revision`
label of `ghcr.io/aunefyren/treningheten:beta`, and the build job runs only if all hold:

- the label is missing or unreadable, or differs from HEAD (unchanged: skip);
- something besides `docs/**`, `*.md` and `.github/ISSUE_TEMPLATE/**` changed since then
  (the published revision not being in history means build);
- `go.yml` has a successful push run for HEAD. A running or red `main` holds the beta back
  and is re-checked next hour; it never falls back to an older green commit.

The reason is written to the run's step summary. A skipped hour ends green.

The build checks out exactly the gated commit and pushes two tags: `beta` (moves, for
`image: aunefyren/treningheten:beta` in compose) and the immutable `beta-<short sha>` (the
same string as the in-app version). To roll back, pin `image: aunefyren/treningheten:beta-<sha>`.

## Changing workflows: `.github/potential-workflows/`

The agent GitHub token has no `workflow` scope. GitHub rejects any push that creates,
changes or deletes a file under `.github/workflows/`, even on a feature branch. So agents
**never touch `.github/workflows/`**. They stage workflow changes instead, and the maintainer
moves them into place within the PR.

### For agents

1. Write the **complete** workflow file, not a fragment or patch, to
   `.github/potential-workflows/<name>.yml`, using the exact file name it should have under
   `.github/workflows/`. For a change to an existing workflow, copy the current file and
   edit the copy.
2. GitHub only runs files in `.github/workflows/`, so staged files never execute. CI on the
   PR still runs the *old* workflows. Say so in the PR if the change can only be verified
   after the move. Check the YAML locally if you can (`actionlint` when available).
3. Update the docs as if the move had already happened: this file, and any feature doc
   that refers to the workflow.
4. Put a **"Workflow files to move"** section at the top of the PR body, with the exact
   commands:

   ```bash
   git mv -f .github/potential-workflows/go.yml .github/workflows/go.yml   # changed
   git mv .github/potential-workflows/beta.yml .github/workflows/beta.yml  # new
   git rm .github/workflows/old.yml                                        # deleted
   ```

   To delete a workflow, list the `git rm` command in that section. Don't stage anything
   for it.
5. Don't leave `.github/potential-workflows/` populated on `main`. After the maintainer's
   move, the directory is empty again and git drops it.

### For the maintainer

Check out the PR branch, run the commands from the PR body, commit (e.g. `chore: move
staged workflows into place`) and push. Then CI runs the new workflows on the PR before
merge.
