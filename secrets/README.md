# Secrets (local only)

Put a GitHub fine-grained PAT in `github_pat` (one line, no quotes):

- Repository access: this private FlashDock repo only
- Permissions: Contents Read (+ enough to read Releases)

Or set environment variable instead:

```bash
export FLASHDOCK_GITHUB_TOKEN=github_pat_xxx
```

Never commit `github_pat`. This directory is gitignored except this README.

Installed app (非开发目录启动) 也可把 PAT 放到：

```text
~/.flashdock/github_pat
```

After rotating a leaked token, replace the local file and any GitHub Actions secret named `FLASHDOCK_GITHUB_TOKEN`.
