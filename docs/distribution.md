# Private Distribution via GitHub Packages

This document describes the current private distribution flow for `@reld/pi-memory`.

## Current approach

The package is distributed as a private npm-style package through **GitHub Packages**, not via a git clone of the source repository.

Why:
- Pi git installs clone the full repository
- we want to distribute the package artifact, not the full development repo
- the published package includes runtime TypeScript extension files plus the compiled backend binary
- the published package excludes Go backend source

Current package name:

```text
@reld/pi-memory
```

Current packaged backend target:

```text
darwin-arm64
```

## Release flow

The intended release flow is:

1. push source changes to the private GitHub repo
2. create and push a tag such as `v0.1.0`
3. GitHub Actions runs validation:
   - `npm ci`
   - `npm run typecheck`
   - `npm run build`
   - `npm pack --dry-run`
4. GitHub Actions publishes `@reld/pi-memory` to GitHub Packages

## Work-machine install prerequisites

The target machine needs npm configured for:
- the `@reld` scope
- GitHub Packages authentication

### 1. Create a GitHub token

Use a token that can read private packages.

Minimum expected capability:
- `read:packages`

If your setup also needs access to a private repository during related workflows, you may additionally need:
- `repo`

### 2. Configure npm to use GitHub Packages for `@reld`

Add this to `~/.npmrc` on the work machine:

```text
@reld:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN
```

You can also set the token through environment/config management if you do not want it stored directly in the file.

### 3. Verify npm can see the package

A useful sanity check is:

```bash
npm view @reld/pi-memory --registry=https://npm.pkg.github.com
```

If auth and registry configuration are correct, npm should be able to resolve the package metadata.

## Pi install flow

Once npm auth is configured, install the package in Pi with:

```bash
pi install npm:@reld/pi-memory@0.1.0
```

You can omit the version to track the latest published release for that registry source:

```bash
pi install npm:@reld/pi-memory
```

For early private testing, installing a pinned version is recommended.

## Update flow

For intentional updates on the work machine, prefer installing the next explicit version:

```bash
pi install npm:@reld/pi-memory@0.1.1
```

This keeps the upgrade path explicit while the package is still in private testing.

## Runtime expectations

The published package contains:
- `extensions/`
- `src/extension/`
- `resources/bin/darwin-arm64/pi-memory-backend`
- docs and package metadata

The published package does not contain:
- `go/`
- development/release-only scripts
- local `dist/` output
- project planning/tracking notes

Runtime backend resolution remains:
1. `PI_MEMORY_BACKEND_PATH`
2. packaged `resources/bin/<platform>-<arch>/<binary>`
3. local `dist/package/bin/<binary>`

In the published package, the packaged `resources/bin/...` path is the intended normal runtime path.

## Troubleshooting

### Package cannot be found

Check:
- the package version exists in GitHub Packages
- your `~/.npmrc` scope mapping is correct
- your token can read private packages

### Pi cannot install the package

Check npm directly first:

```bash
npm view @reld/pi-memory --registry=https://npm.pkg.github.com
```

If npm cannot resolve the package, Pi will not be able to install it either.

### Backend binary cannot be found at runtime

Check:
- the published package includes `resources/bin/darwin-arm64/pi-memory-backend`
- the target machine is actually `darwin-arm64`
- `PI_MEMORY_BACKEND_PATH` is not pointing somewhere invalid
