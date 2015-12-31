Feedback, bug reports, and pull requests are greatly appreciated :)

### Issues

The following are all great reasons to submit an issue:

1. You found a bug in the code.
2. Something is missing from the documentation or the existing documentation is unclear.
3. You have an idea for a new feature.

If you are thinking about submitting an issue please remember to:

1. Describe the issue in detail.
2. If applicable, describe the steps to reproduce the error, which probably should include some example code.
3. Mention details about your platform: OS, version of Go and Redis, etc.

### Pull Requests

Zoom uses semantic versioning and the [git branching model described here](http://nvie.com/posts/a-successful-git-branching-model/).
If you plan on submitting a pull request, you should:

1. Fork the repository.
2. Create a new "feature branch" off of **develop** (not master) with a descriptive name (e.g. fix-database-error).
3. Make your changes in the feature branch.
4. Run the tests to make sure that they still pass. Updated the tests if needed.
5. Submit a pull request to merge your feature branch into the **develop** branch. Please do not request to merge directly into master.

### Third-Party Dependencies

Zoom uses [Glide](https://github.com/Masterminds/glide) to manage dependencies.
If you update or add any new dependencies, make sure you edit glide.yaml
appropriately. All dependencies should be locked to a specific version in
glide.yaml.

1. If the project supports semantic versioning and the current version is
   greater than or equal to 1.0, the version should be pinned to the current
   major version. E.g., `version: 2.x`.
2. If the project supports semantic versioning and the current version is less
	than 1.0, the version should be pinned to the latest patch. E.g.,
	`version: 0.4.2`.
3. If the project does not support semantic versioning, then the version should
   be pinned to the latest commit hash. E.g.,
   `ref: 2b2c4ccb8692bb9d0ac6411c1fe47bb04be0ee05`.

After you have added or updated the dependency, run `glide install` to have
Glide get the appropriate version and put it in the `vendor` folder.

For Zoom, we commit all third-party dependencies in the `vendor` folder to
version control. That way, you don't have to use Glide (or even know that Glide
exists) in order to use Zoom. In order to get this to work properly, you must
remove the `.git` file from the project in the `vendor` folder. If you don't do
this, git will either ignore the directory where the dependency was installed or
use submodules, which is not desirable (the exact behavior depends on how you
added the directories). After you have removed the `.git`, run `git add` as you
would normally.
