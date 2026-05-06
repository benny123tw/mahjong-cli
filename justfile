# mahjong-cli — task runner
#
# Run `just` to see the available recipes.

# bumpkin drives semver tagging from conventional commits. Override with
# `just BUMPKIN=bumpkin <recipe>` if you have it installed on PATH.
BUMPKIN := "go run github.com/benny123tw/bumpkin/cmd/bumpkin@latest"

# Default recipe — list everything.
default:
    @just --list

# Build the binary into ./bin/mahjong.
build:
    go build -o ./bin/mahjong ./cmd/mahjong

# Run the full test suite.
test:
    go test -buildvcs=false ./...

# Run tests with coverage and open the HTML report.
cover:
    go test -buildvcs=false -coverprofile=cover.out ./...
    go tool cover -html=cover.out

# Update golden test files (regenerate from current code output).
test-update:
    go test -buildvcs=false ./cmd/... -update

# Lint the codebase.
lint:
    golangci-lint run

# Lint and apply auto-fixes where possible.
lint-fix:
    golangci-lint run --fix

# Format Go code (gofmt + gofumpt + golines via the linter formatters).
fmt:
    golangci-lint fmt

# Run go vet over the project.
vet:
    go vet ./...

# Verify the project: format check, lint, vet, and tests.
verify: fmt lint vet test

# Tidy go.mod and go.sum.
tidy:
    go mod tidy

# Launch the play TUI (Unicode tile rendering).
play *args:
    go run -buildvcs=false ./cmd/mahjong play {{args}}

# Launch the play TUI with ASCII tile rendering.
play-ascii *args:
    go run -buildvcs=false ./cmd/mahjong play --ascii {{args}}

# Run `mahjong calc <hand>` directly.
calc hand *args:
    go run -buildvcs=false ./cmd/mahjong calc "{{hand}}" {{args}}

# Remove build artifacts.
clean:
    rm -rf ./bin ./cover.out ./dist

# Show the current version (latest git tag).
current:
    @{{BUMPKIN}} current

# Preview the next release without tagging or pushing.
release-dry:
    {{BUMPKIN}} --conventional --dry-run --yes

# Cut a release from conventional commits: bump, tag, push (CI does the rest).
release:
    {{BUMPKIN}} --conventional

# Force a patch bump.
release-patch:
    {{BUMPKIN}} --patch

# Force a minor bump.
release-minor:
    {{BUMPKIN}} --minor

# Force a major bump.
release-major:
    {{BUMPKIN}} --major

# Local goreleaser snapshot build (no publish, lands in ./dist).
release-snapshot:
    goreleaser release --snapshot --clean --skip=publish
