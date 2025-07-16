package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dominikbraun/graph"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/ast"
)

func main() {
	// Command line flags
	var (
		taskfileURL = flag.String("taskfile", "https://raw.githubusercontent.com/gkwa/ringgem/refs/heads/master/Taskfile.yaml", "Taskfile URL or path")
		startTask   = flag.String("start", "default", "Task to start dependency tree from")
		noCache     = flag.Bool("no-cache", false, "Force download without using cache")
	)
	flag.Parse()

	// Enable remote Taskfiles experiment - need to parse experiments first
	os.Setenv("TASK_X_REMOTE_TASKFILES", "1")

	// Parse experiments with current directory
	experiments.Parse(".")

	// Validate experiments
	if err := experiments.Validate(); err != nil {
		panic(fmt.Sprintf("Failed to validate experiments: %v", err))
	}

	// Create a root node for the Taskfile
	node, err := taskfile.NewRootNode(*taskfileURL, "", false, 30*time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to create root node: %v", err))
	}

	// Create a reader with remote-specific options
	reader := taskfile.NewReader(
		taskfile.WithInsecure(false),    // Don't allow HTTP (only HTTPS)
		taskfile.WithDownload(*noCache), // Force download if no-cache is set
		taskfile.WithOffline(false),     // Allow network requests
		taskfile.WithTempDir(os.TempDir()),
		taskfile.WithCacheExpiryDuration(24*time.Hour),
		taskfile.WithDebugFunc(func(msg string) {
			fmt.Printf("DEBUG: %s\n", msg)
		}),
		taskfile.WithPromptFunc(func(prompt string) error {
			fmt.Printf("PROMPT: %s\n", prompt)
			// Auto-accept prompts for demo purposes
			// In production, you'd want to prompt the user
			return nil
		}),
	)

	// Read the Taskfile graph (including remote includes)
	taskfileGraph, err := reader.Read(context.Background(), node)
	if err != nil {
		panic(fmt.Sprintf("Failed to read Taskfile: %v", err))
	}

	// Get the merged Taskfile
	mergedTaskfile, err := taskfileGraph.Merge()
	if err != nil {
		panic(fmt.Sprintf("Failed to merge Taskfile: %v", err))
	}

	fmt.Printf("=== Taskfile Graph Analysis ===\n")
	fmt.Printf("Location: %s\n", mergedTaskfile.Location)
	fmt.Printf("Version: %s\n", mergedTaskfile.Version.String())
	fmt.Printf("\n")

	// Traverse the Taskfile inclusion graph
	fmt.Printf("=== Taskfile Inclusion Graph ===\n")
	hashes, err := graph.TopologicalSort(taskfileGraph.Graph)
	if err != nil {
		panic(fmt.Sprintf("Failed to sort graph: %v", err))
	}

	for i, hash := range hashes {
		vertex, err := taskfileGraph.Vertex(hash)
		if err != nil {
			continue
		}
		fmt.Printf("%d. Taskfile: %s\n", i+1, vertex.URI)

		// Show includes for this Taskfile
		if vertex.Taskfile.Includes.Len() > 0 {
			fmt.Printf("   Includes:\n")
			for namespace, include := range vertex.Taskfile.Includes.All() {
				fmt.Printf("     - %s: %s\n", namespace, include.Taskfile)
			}
		}
	}
	fmt.Printf("\n")

	// Analyze task dependencies
	fmt.Printf("=== Task Dependencies ===\n")
	buildTaskDependencyGraph(mergedTaskfile)

	for taskName, task := range mergedTaskfile.Tasks.All(nil) {
		fmt.Printf("Task: %s", taskName)
		if task.Desc != "" {
			fmt.Printf(" - %s", task.Desc)
		}
		fmt.Printf("\n")

		if len(task.Deps) > 0 {
			fmt.Printf("  Dependencies:\n")
			for _, dep := range task.Deps {
				fmt.Printf("    - %s\n", dep.Task)
			}
		}

		if len(task.Cmds) > 0 {
			fmt.Printf("  Commands:\n")
			for _, cmd := range task.Cmds {
				if cmd.Cmd != "" {
					fmt.Printf("    - cmd: %s\n", cmd.Cmd)
				}
				if cmd.Task != "" {
					fmt.Printf("    - task: %s\n", cmd.Task)
				}
			}
		}
		fmt.Printf("\n")
	}

	// Show complete dependency tree from starting task
	fmt.Printf("=== Complete Dependency Tree from '%s' task ===\n", *startTask)
	if _, exists := mergedTaskfile.Tasks.Get(*startTask); exists {
		showDependencyTree(mergedTaskfile, *startTask, 0)
	} else {
		fmt.Printf("Task '%s' not found\n", *startTask)
		fmt.Printf("Available tasks:\n")
		for taskName := range mergedTaskfile.Tasks.All(nil) {
			fmt.Printf("  - %s\n", taskName)
		}
	}
}

// buildTaskDependencyGraph creates a dependency map for tasks
func buildTaskDependencyGraph(tf *ast.Taskfile) map[string][]string {
	deps := make(map[string][]string)

	for taskName, task := range tf.Tasks.All(nil) {
		var taskDeps []string

		// Add explicit dependencies
		for _, dep := range task.Deps {
			taskDeps = append(taskDeps, dep.Task)
		}

		// Add task calls from commands
		for _, cmd := range task.Cmds {
			if cmd.Task != "" {
				taskDeps = append(taskDeps, cmd.Task)
			}
		}

		deps[taskName] = taskDeps
	}

	return deps
}

// showDependencyTree shows the complete dependency tree without tracking visited nodes
func showDependencyTree(tf *ast.Taskfile, taskName string, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	task, exists := tf.Tasks.Get(taskName)
	if !exists {
		fmt.Printf("%s%s (not found)\n", indent, taskName)
		return
	}

	fmt.Printf("%s%s", indent, taskName)
	if task.Desc != "" {
		fmt.Printf(" - %s", task.Desc)
	}
	fmt.Printf("\n")

	// Show all dependencies recursively
	for _, dep := range task.Deps {
		showDependencyTree(tf, dep.Task, depth+1)
	}

	// Show all task calls from commands recursively
	for _, cmd := range task.Cmds {
		if cmd.Task != "" {
			showDependencyTree(tf, cmd.Task, depth+1)
		}
	}
}
