# write python script exec the following
# add important comment to function and useful location
# add an --dry-run command args, so that nothing will execute, and important debug message will be print out
# 1. read cmd/qbt/cmd/root.go
# 2. find a version string in format ```const VERSION = "0.1.122"```
# 3. change this line to ```const VERSION = "0.1.123"```
# 4. change README.md  "go install  github.com/qbtrade/qbt@v0.1.122" to "go install  github.com/qbtrade/qbt@v0.1.123"
# 4. exec git commit -a -m "add to version 0.1.123"
# 5. exec git tag v0.1.123
# 6. exec git push
# note that the version may be 0.1.** or 0.11.** or 1.0.** or 1.1.*** etc
# note the version number MUST  a.b.c, where a, b, c are all integers
# note that after 0.1.9 it will be 0.1.10
# note that the version number is not necessary to be 3 digits, it can be 1/2/3/4/5 digits

import argparse
import re
import subprocess


def parse_arguments():
    parser = argparse.ArgumentParser(description="Update qbtrade/qbt version.")
    parser.add_argument("--dry-run", action="store_true", help="Run the script without executing any command.")
    return parser.parse_args()


def read_file(file_path):
    with open(file_path, "r") as file:
        return file.read()


def write_file(file_path, content):
    with open(file_path, "w") as file:
        file.write(content)


def update_version(version_str):
    major, minor, patch = map(int, version_str.split("."))
    patch += 1
    return f"{major}.{minor}.{patch}"


def exec_command(cmd, dry_run):
    if dry_run:
        print(f"Dry run: {cmd}")
    else:
        subprocess.run(cmd, shell=True, check=True)


def main():
    args = parse_arguments()
    dry_run = args.dry_run

    # Read and update root.go
    root_go_path = "cmd/qbt/cmd/root.go"
    root_go_content = read_file(root_go_path)
    version_regex = r'(const VERSION = ")(\d+\.\d+\.\d+)(")'
    new_version = update_version(re.search(version_regex, root_go_content).group(2))
    new_root_go_content = re.sub(version_regex, fr'\g<1>{new_version}\g<3>', root_go_content)
    write_file(root_go_path, new_root_go_content)

    # Update README.md
    readme_path = "README.md"
    readme_content = read_file(readme_path)
    version_regex = r'(qbt@v)(\d+\.\d+\.\d+)'
    new_readme_content = re.sub(version_regex, fr'\g<1>{new_version}', readme_content)
    write_file(readme_path, new_readme_content)

    # Execute git commands
    exec_command(f'git commit -a -m "add to version {new_version}"', dry_run)
    exec_command(f'git tag v{new_version}', dry_run)
    exec_command('git pull', dry_run)
    exec_command('git push', dry_run)
    exec_command('git push --tags', dry_run)


if __name__ == "__main__":
    main()
