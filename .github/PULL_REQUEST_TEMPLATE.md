## Summary

<!-- What does this PR do? One or two sentences. -->

## Related issues

<!-- Link issues this PR closes. Use "Closes #123" or "Fixes #123". -->

Closes #

## Checklist

### All PRs
- [ ] Branch named correctly (`feat/*`, `fix/*`, `refactor/*`, `chore/*`, `docs/*`, `ci/*`)
- [ ] Commit messages follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/)
- [ ] Code builds: `go build ./...`
- [ ] Tests pass: `go test ./...`
- [ ] No new `go vet` warnings
- [ ] No new staticcheck warnings
- [ ] No new ESLint or TypeScript errors in `web/`

### Go changes (`**/*.go`, `go.mod`, `go.sum`, `migrations/`)
- [ ] Database migrations tested up and down
- [ ] New code has tests where it makes sense
- [ ] Imports grouped (stdlib → third-party → internal)

### Frontend changes (`web/**`)
- [ ] `npm run lint` passes
- [ ] `npx tsc -b --noEmit` passes
- [ ] `npm run build` succeeds

### Breaking changes
- [ ] Called out in PR description so release notes catch it
- [ ] Migration path documented for existing users

### Screenshots / demo

<!-- If the PR changes UI, paste before/after screenshots. If it changes bot behavior, paste a sample conversation. -->
