# Contributing

Thank you for your help. This file tells you how to make a change.

## Before you start

- Find an issue for the change, or open a new issue. Each milestone has a tracking issue with the label `milestone`.
- For a large change, write your plan in the issue first. Then the maintainers can tell you if the plan fits the specification in `spec/`.

## Make a change

1. Make a branch from `main`.
2. Make the change. Keep each pull request on one topic.
3. Run the checks:

   ```sh
   make fmt vet test lint
   ```

4. Push the branch and open a pull request. Fill in the template.
5. Add the labels `type: ...` and `area: ...`, and the milestone.

## Rules for code

- Format the code with `gofmt`.
- Write a test for each new function that has logic.
- Do not add a dependency without a reason in the pull request.
- Pin each GitHub Action to a commit SHA, with the tag in a comment.

## Rules for text

Write in plain English. Use short sentences and the active voice. Use one word for one meaning. Do not break a sentence over two lines. Use the same style in the code comments, the documents, the commit messages and the pull requests.

## Sign your commits

Add a `Signed-off-by` line to each commit (`git commit -s`). The line says that you agree to the [Developer Certificate of Origin](https://developercertificate.org/).
