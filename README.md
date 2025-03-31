# gh-prview

A GitHub CLI extension to render pull requests in the terminal with comments and reviews in chronological order, similar to the web view.

## Installation

```bash
gh extension install bmon/gh-prview
```

## Usage

```bash
# Show the current branch's pull request
gh prview

# Show a specific pull request by number
gh prview 123
```

You might like to use a pager like `less` when viewing the output.

## Roadmap

- Add support for color
- Add support for reactions
- Support for alternative formatting options
