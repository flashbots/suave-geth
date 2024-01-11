suave-geth Release Checklist
============================

Note: the best days to release a new version are Monday to Wednesday. Never release on a Friday (because of the risk of needing to work on the weekend).

### Prepare the release

- [ ] Run linter and tests:
  ```bash
  $ make lint && make test && make suave
  $ ./build/bin/suave-geth version
  ```
- [ ] Test latest version with [suapp-examples](https://github.com/flashbots/suapp-examples) and [suave-std](https://github.com/flashbots/suave-std)

### Update documentation (if needed)

Prepare documentation updates before the release is published. Only if needed.

- [ ] [Docs](https://github.com/flashbots/suave-docs)
- [ ] [Specs](https://github.com/flashbots/suave-specs)

### Publish the release

- [ ] Pick the version (i.e. `v0.2.0-stable`)
- [ ] Update the version number in [`params/version.go`](../params/version.go)
- [ ] Make a commit with the version number change (i.e. `git commit -m 'bump version to v0.2.0'`)
- [ ] Tag new version (`git tag -s v0.2.0`)
- [ ] Push tag to Github. At this point, [CI](https://github.com/flashbots/suave-geth/blob/release-checklist/.github/workflows/releaser.yml) builds the packages, publishes Docker images to [Docker hub](https://hub.docker.com/r/flashbots/suave-geth), and creates a draft release on Github (all using [Goreleaser](https://github.com/flashbots/suave-geth/blob/release-checklist/.goreleaser.yaml))
- [ ] Edit the draft release on Github to prepare nice release notes
- [ ] Publish the release (note: this will send an email to subscribers on Github)

### After publishing

- [ ] Test the Release: Download the release binary: `curl -L https://suaveup.flashbots.net | bash` and check the version with `suave-geth version`
- [ ] Increment the version number to the next patch version and `-dev` meta in [`params/version.go`](../params/version.go) and push a commit

### Announcing

- [ ] Make a forum post, possibly announce on Discord and Twitter
