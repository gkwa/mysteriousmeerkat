# Taskfile Graph Demo

This demo shows how to use go-task as a library to load and traverse Taskfile graphs, including support for remote Taskfiles.

## Features

- Load Taskfiles from local files or remote URLs (HTTP/HTTPS, Git)
- Traverse the Taskfile inclusion graph
- Analyze task dependencies
- Perform DFS traversal on task dependency graphs
- Support for remote Taskfiles with security features

## Usage

```bash
# Enable remote Taskfiles experiment
export TASK_X_REMOTE_TASKFILES=1

# Run the demo
go run main.go
