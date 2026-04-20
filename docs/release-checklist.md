# First Private Release Checklist

This checklist is for the first real private release of `@reld/pi-memory` through GitHub Packages.

## Release model

Current release/versioning model:
- source of truth: private GitHub repo
- package registry: GitHub Packages
- package name: `@reld/pi-memory`
- target runtime binary shipped in package: `darwin-arm64`
- release trigger: git tag such as `v0.1.0`
- install target on another machine: pinned package version via `pi install npm:@reld/pi-memory@<version>`

Version/tag convention:
- package version in `package.json`: `0.1.0`
- git tag for the release: `v0.1.0`

Those should match.

## Before creating the tag

### Code/package readiness

- [ ] `package.json` version is correct
- [ ] package name is `@reld/pi-memory`
- [ ] published `files` list still matches intended runtime contents
- [ ] local build or CI build will produce `resources/bin/darwin-arm64/pi-memory-backend` for the package publish
- [ ] README/distribution docs reflect the current install flow

### Local validation

Run:

```bash
vp run build
vp run pack:dry-run
```

Confirm:
- [ ] typecheck passes
- [ ] backend build passes
- [ ] `npm pack --dry-run` shows the expected package contents
- [ ] Go source is not included in the package
- [ ] packaged backend binary is included in the dry-run package output

### GitHub workflow readiness

- [ ] `.github/workflows/ci.yml` is present and correct
- [ ] `.github/workflows/publish.yml` is present and correct
- [ ] GitHub Packages publish permissions are available for the repo/account

## Tag and publish

Commit everything needed for the release, then create and push the tag:

```bash
git tag v0.1.0
git push origin main --tags
```

Expected result:
- [ ] the publish workflow starts for the tag
- [ ] workflow completes successfully
- [ ] package version `0.1.0` appears in GitHub Packages for `@reld/pi-memory`

## Work-machine prerequisites

On the target machine:

### npm auth setup

Add to `~/.npmrc`:

```text
@reld:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN
```

Check:
- [ ] GitHub token has `read:packages`
- [ ] npm scope registry is configured for `@reld`

### Package visibility check

Run:

```bash
npm view @reld/pi-memory --registry=https://npm.pkg.github.com
```

Check:
- [ ] npm can resolve the package metadata

## Install in Pi on the work machine

Install the pinned package version:

```bash
pi install npm:@reld/pi-memory@0.1.0
```

Check:
- [ ] Pi installs the package successfully
- [ ] the extension loads in Pi
- [ ] no package-resolution/auth errors occur

## Runtime smoke test on the work machine

Inside a real project:

- [ ] run `/pi-memory-init`
- [ ] run `/pi-memory-status`
- [ ] verify the backend binary is found automatically
- [ ] verify no manual `PI_MEMORY_BACKEND_PATH` override is needed on `darwin-arm64`
- [ ] verify session ingestion works
- [ ] verify recall/search commands work

## If something fails

Useful checks:

```bash
npm view @reld/pi-memory --registry=https://npm.pkg.github.com
pi list
```

If runtime cannot find the backend:
- verify the installed package contains `resources/bin/darwin-arm64/pi-memory-backend`
- verify the work machine is actually `darwin-arm64`
- verify `PI_MEMORY_BACKEND_PATH` is unset or valid

## After the first successful release

- [ ] record the outcome in `VALIDATION.md`
- [ ] update `HANDOFF.md`
- [ ] decide whether to create a GitHub Release entry in addition to package publication
- [ ] decide when to broaden distribution beyond private GitHub Packages
