# AGENTS.md

## Guide for Agents and Contributors

This document defines standards and practices for automated agents and humans contributing to this project.

---

### 1. Commit Standards
- Use descriptive commit messages in the format [type]: short description (e.g., `fix: fix transaction validation`).
- Commits should be atomic: each commit addresses a single intent or issue.

### 2. Tests
- Every relevant change must be covered by automated tests.
- Always run the tests before committing or opening a PR.
- The standard command to run the tests is:
  ```sh
  npm run test
  ```
- If a test fails after a change, adjust the code or the test to ensure all tests pass.

### 3. Traceability and Metadata
- Whenever possible, include traceability information (e.g., error location, origin context) in processed data.
- Use fields like `meta.location` to indicate the origin of data or errors.

### 4. Interoperability
- New agents, validators, or parsers must follow the data structure and metadata standards already established in the project.
- Avoid reinventing existing conventions; consult this document and the codebase for standards.

### 5. General Principles
- Prefer clarity and traceability over "magic" or premature optimizations.
- When in doubt, run the tests and follow the commit and metadata standards.
- Document non-trivial decisions in this file.

---

> This document should be kept up to date as the project evolves. 