# Rules

* [Secrets stay out of code, logs and output](secrets.md) - Credentials come from the environment or an ignored file, and never appear in code, logs, test output, commits or this bundle.
* [Commits are atomic and conventional](commits.md) - One logical change per commit, written as a Conventional Commit, with tests passing at every commit.
* [Work on a branch, merge through a pull request](branches.md) - Agents commit and push on their own branch without asking; the default branch changes only through a pull request the owner merges.
* [Changes come with tests](tests.md) - New behaviour comes with a test, a bug fix with a test that fails without the fix, and unit tests do not touch real services.
