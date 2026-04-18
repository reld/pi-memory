# AGENTS

## Pi.dev Repository
https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent

## Pi.dev DOCS
This is the Pi documentation. Always reference this when we are creating extensions, skills, pi packages etc...
https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent#readme

## Project notes
- This project is being built as a **Pi package** for distribution.
- The package will include both a **TypeScript Pi extension** and a **Go backend binary**.
- For TypeScript project setup and workflow, prefer **VitePlus**.
- VitePlus is already installed on this system.
- VitePlus docs: https://viteplus.dev/guide/

## VitePlus usage rules
- Prefer `vp` for JavaScript/TypeScript project workflows instead of raw `npm` when working in this repo.
- Use `vp add <pkg>` to add dependencies.
- Use `vp remove <pkg>` to remove dependencies.
- Use `vp update` / `vp outdated` / `vp why` / `vp info` for dependency management.
- Use `vp install` instead of `npm install` unless there is a strong reason not to.
- Use `vp exec <bin>` for local project binaries instead of `npx <bin>` when possible.
- Use `vp check` for the standard format/lint/type-check workflow when applicable.
- Use `vp run <script>` (or `vpr <script>`) for package.json scripts instead of `npm run <script>` when appropriate.
- Only fall back to raw `npm`/`npx` if `vp` cannot do the needed task or if there is a clear project-specific reason.

