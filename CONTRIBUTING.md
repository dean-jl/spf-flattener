# Contributing to SPF Flattener

Thanks for your interest in contributing! This project is designed to be fork-friendly and extensible, with a focus on maintainable architecture and safe defaults. Whether you're fixing a bug, adding a new DNS provider, or improving documentation—your input is welcome.

## 🛠 Fork-Friendly by Design

- Modular architecture makes it easy to extend.
- Adding new DNS providers is straightforward (see `internal/` and `docs/`).
- Dry-run mode ensures safe experimentation.

Feel free to fork this repository and adapt it to your needs. If you build something useful or add support for a new provider, I’d love to hear about it.

## 📬 Pull Requests Welcome (with Caveats)

You’re welcome to submit pull requests, but please note:
- This project is maintained as a reference implementation and personal utility.
- I may not be able to actively review or merge contributions.
- Contributions may be acknowledged or incorporated at my discretion.

If you'd like to collaborate more deeply or help maintain the project, feel free to open a discussion or issue to introduce yourself.

## 🧭 Contribution Guidelines

Before submitting a pull request:
- Run `go fmt ./...` to format code.
- Run `go vet ./...` to check for issues.
- Run `go test ./...` to ensure all tests pass.
- Include a brief description of what your change does and why it’s useful.

### For New DNS Providers

If you're adding support for a new provider:
- Implement the required interface methods (`Ping`, `RetrieveRecords`, `CreateRecord`, `UpdateRecord`, `DeleteRecord`).
- Follow the existing error handling patterns.
- Include integration test instructions or sample config usage.
- Update documentation in `docs/` and `README.md` as needed.

## 🧪 Testing Philosophy

This project uses a hybrid testing approach:
- **Automated unit tests** verify core logic.
- **Manual verification** ensures real-world CLI behavior.

See the `README.md` for detailed testing instructions.

## 🙌 Thank You

Your interest and effort are appreciated. Even if your contribution isn’t merged, your ideas help improve the tool and its ecosystem.
