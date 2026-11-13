# Releasing LGTMFaster

Pushing a version tag such as `v1.0.0` starts `.github/workflows/release.yml`, which:

1. runs `go vet` and the tests (`go test -race ./...`),
2. builds `lgtmfaster` for Linux, macOS and Windows on amd64 and arm64 (`CGO_ENABLED=0`, stripped,
   with the version, commit and build date injected into `internal/version`),
3. publishes the archives, `checksums.txt` (SHA256) and generated release notes as a GitHub Release,
4. updates the Homebrew cask in the `jaforsgren/homebrew-tap` repository.

Everything uses free tools: GitHub Actions on a public repository and GoReleaser OSS
(`.goreleaser.yaml`), with no Docker.

## One-time setup

### 1. Create the Homebrew tap

Create a **public** repository named exactly `homebrew-tap` under the `jaforsgren` account
(`https://github.com/jaforsgren/homebrew-tap`), with a `main` branch (tick "Add a README" so the
branch exists). Homebrew finds it as the tap `jaforsgren/tap`. GoReleaser writes the cask to
`Casks/lgtmfaster.rb` there on every release; nothing else has to be added by hand.

### 2. Create a token for the tap

The workflow's built-in `GITHUB_TOKEN` can only write to `LGTMFaster` itself, so pushing to the tap
needs its own token. Create a **fine-grained personal access token** on the `jaforsgren` account
(Settings → Developer settings → Personal access tokens → Fine-grained tokens):

- **Resource owner:** `jaforsgren`
- **Repository access:** Only select repositories → `homebrew-tap`
- **Repository permissions:** Contents → **Read and write** (Metadata → Read-only is added
  automatically). Nothing else.
- **Expiration:** pick one and note the date; the tap update fails once it expires (the GitHub
  Release itself is still published).

### 3. Add the repository secret

In `jaforsgren/LGTMFaster` → Settings → Secrets and variables → Actions → New repository secret:

| Secret | Value |
|---|---|
| `HOMEBREW_TAP_GITHUB_TOKEN` | the token from step 2 |

`GITHUB_TOKEN` is provided by GitHub Actions automatically; the release job is granted
`contents: write` for it in the workflow, every other job only `contents: read`.

### 4. Check the Actions settings

In Settings → Actions → General, Actions must be allowed to run (the default) and allowed to use
actions from `actions/*` and `goreleaser/*`.

## Testing the release locally

Install GoReleaser (`brew install goreleaser`), then from the repository root:

```bash
make release-check   # goreleaser check: validates .goreleaser.yaml
make snapshot        # goreleaser release --snapshot --clean
```

The snapshot builds every archive into `dist/` (ignored by git) with a version like
`1.0.1-snapshot` and publishes nothing: no release, no tap update, no token needed. Check that:

- `dist/` holds a `.tar.gz` for Linux and macOS and a `.zip` for Windows, for amd64 and arm64,
- `dist/checksums.txt` lists all of them,
- the binary for your machine runs (it is in the `dist/lgtmfaster_<os>_<arch>*` folder),
- the generated cask, `lgtmfaster.rb` somewhere under `dist/`, looks right (it is generated but
  not pushed).

To try the whole pipeline without a real version, push a pre-release tag such as `v0.1.0-rc1`: it
is published as a GitHub pre-release and the tap is skipped (`skip_upload: auto`). Delete the
pre-release and its tag afterwards if you do not want to keep them.

## Publishing a new version

1. Make sure `main` is green and holds what should be released.
2. Pick the next [semantic version](https://semver.org) and tag the commit:

   ```bash
   git checkout main && git pull
   git tag -a v1.0.0 -m "v1.0.0"
   git push origin v1.0.0
   ```

3. Follow the run under the repository's **Actions** tab. When it finishes, the release is on the
   **Releases** page and `Casks/lgtmfaster.rb` in the tap points at it.

Release notes are generated from the commits since the previous tag; `docs:`, `test:`, `style:`
and `chore:` commits are left out.

### Duplicate and conflicting releases

- Only one release runs at a time (`concurrency: release`); a second tag waits for the first.
- Tags must be semantic versions; GoReleaser rejects anything else.
- Re-running a release for a tag that already has a GitHub Release fails when it tries to upload
  assets that already exist, instead of overwriting them. To redo a release, delete the GitHub
  Release (and, if the tag must point elsewhere, the tag) first.
- If the tests fail, nothing is published.

## Installing

### Homebrew (macOS, Intel and Apple Silicon)

```bash
brew install --cask jaforsgren/tap/lgtmfaster
```

Upgrade with `brew upgrade --cask lgtmfaster`. Homebrew picks the build for the Mac's architecture.

The binaries are not signed by Apple (that needs a paid developer account). The cask removes the
macOS quarantine flag after installing, as GoReleaser recommends for unsigned casks; without that
macOS would refuse to run the downloaded binary.

### Standalone executables

Download the archive for your platform from the
[Releases page](https://github.com/jaforsgren/LGTMFaster/releases):

| Platform | Archive |
|---|---|
| macOS, Apple Silicon | `lgtmfaster_<version>_darwin_arm64.tar.gz` |
| macOS, Intel | `lgtmfaster_<version>_darwin_amd64.tar.gz` |
| Linux, x86-64 | `lgtmfaster_<version>_linux_amd64.tar.gz` |
| Linux, ARM64 | `lgtmfaster_<version>_linux_arm64.tar.gz` |
| Windows, x86-64 | `lgtmfaster_<version>_windows_amd64.zip` |
| Windows, ARM64 | `lgtmfaster_<version>_windows_arm64.zip` |

Verify the download against `checksums.txt` from the same release, then unpack it and put
`lgtmfaster` somewhere on your `PATH`:

```bash
sha256sum --check --ignore-missing checksums.txt   # macOS: shasum -a 256 --check --ignore-missing checksums.txt
tar -xzf lgtmfaster_<version>_linux_amd64.tar.gz
sudo mv lgtmfaster /usr/local/bin/
```

On macOS a binary downloaded with a browser is quarantined; clear it with
`xattr -d com.apple.quarantine lgtmfaster`, or install with Homebrew instead.
