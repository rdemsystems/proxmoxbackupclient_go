# Contributing to Proxmox Backup Guardian GUI

Thank you for your interest in contributing! This project is a GUI wrapper for the excellent [proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) CLI tool.

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Fyne dependencies:
  - **Linux**: `gcc`, `libgl1-mesa-dev`, `xorg-dev`
  - **Windows**: MinGW-w64
  - **macOS**: Xcode Command Line Tools

### Development Setup

```bash
# Clone the repository
git clone git@git.pa4.rdem-systems.com:rdem-systems/proxmox-backup-client-go-gui.git
cd proxmox-backup-client-go-gui

# Install dependencies
cd gui
go mod download

# Run the GUI
go run .

# Build
cd ..
./build_gui.sh  # Linux/macOS
# or
./build_gui.bat  # Windows
```

## 🏗️ Project Structure

```
gui/
├── main.go              # Main UI with tabs
├── config.go            # Configuration management
├── backup.go            # Backup execution logic
├── backup_config_ui.go  # Advanced backup UI
├── jobs.go              # Job management
├── jobs_ui.go           # Jobs list UI
├── scheduler.go         # Scheduling (cron/Task Scheduler)
└── go.mod              # Dependencies
```

## 📝 Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Write meaningful commit messages

### Fyne UI Guidelines

- Use Fyne's theming system
- Prefer built-in widgets when possible
- Keep UI responsive (use goroutines for long operations)
- Test on multiple platforms if possible

### Commit Messages

```
type(scope): subject

body (optional)

footer (optional)
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting, missing semi colons, etc.
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

**Examples:**
```
feat(ui): add multi-folder selection

Allow users to select multiple folders for backup instead of just one.

Closes #42
```

```
fix(scheduler): correct cron parsing on Windows

The cron expression was not properly converted to Task Scheduler format.
```

## 🧪 Testing

```bash
cd gui

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

### Manual Testing Checklist

Before submitting a PR, please test:

- [ ] PBS connection test works
- [ ] Folder selection works
- [ ] Exclusion patterns work
- [ ] Job creation succeeds
- [ ] Scheduling is properly set up
- [ ] Export JSON/INI works
- [ ] Import config works
- [ ] UI is responsive on your platform

## 🔄 Pull Request Process

1. **Fork** the repository (or create a branch if you have access)

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write clear, concise code
   - Add tests if applicable
   - Update documentation

4. **Test thoroughly**
   ```bash
   go test ./...
   go build
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

6. **Push to your branch**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Open a Pull Request**
   - Fill out the PR template
   - Link any related issues
   - Request review from maintainers

### PR Review Process

- PRs require at least one approval
- CI/CD must pass (builds, tests, linting)
- Code review feedback should be addressed
- Squash commits if requested

## 🐛 Reporting Bugs

Use the GitLab issue tracker. Include:

- **Description**: Clear and concise description
- **Steps to Reproduce**: Numbered steps
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Environment**: OS, Go version, GUI version
- **Logs**: Relevant error messages or logs

**Example:**

```markdown
## Description
Connection test to PBS fails with certificate error

## Steps to Reproduce
1. Open GUI
2. Configure PBS with valid URL
3. Click "Test Connection"
4. Error appears

## Expected
Connection should succeed

## Actual
Error: "certificate verify failed"

## Environment
- OS: Ubuntu 22.04
- Go: 1.21.5
- GUI: v1.0.0

## Logs
```
2026-03-17 10:30:45 ERROR: certificate verify failed for pbs.example.com
```
```

## 💡 Feature Requests

Feature requests are welcome! Please:

- Check if the feature is already requested
- Explain the use case clearly
- Describe the expected behavior
- Consider implementation complexity

## 🤝 Code of Conduct

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone.

### Our Standards

✅ **Do:**
- Be respectful and inclusive
- Accept constructive criticism
- Focus on what's best for the community

❌ **Don't:**
- Use sexualized language or imagery
- Make derogatory comments or personal attacks
- Harass others publicly or privately

## 📄 License

By contributing, you agree that your contributions will be licensed under the same license as the original project (GPLv3).

## 🙏 Acknowledgments

- Thanks to [tizbac](https://github.com/tizbac) for the original proxmoxbackupclient_go CLI
- Thanks to the Fyne team for the excellent GUI framework
- Thanks to RDEM Systems for sponsoring this GUI development

## 📞 Contact

- **GitLab Issues**: Primary communication channel
- **Email**: contact@rdem-systems.com (for sensitive matters)
- **Website**: https://nimbus.rdem-systems.com

---

**Happy Contributing! 🎉**
