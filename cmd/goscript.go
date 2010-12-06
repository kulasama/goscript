// Copyright 2010  The "goscript" Authors
//
// Use of this source code is governed by the Simplified BSD License
// that can be found in the LICENSE file.
//
// This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
// OR CONDITIONS OF ANY KIND, either express or implied. See the License
// for more details.

package main

import (
	"exec"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

const EXIT_CODE = 2

var ENVIRON []string


// Base to access to "mtime" of given file.
func _time(filename string, mtime int64) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not access: %s\n", err)
		os.Exit(EXIT_CODE)
	}
	
	if mtime != 0 {
		info.Mtime_ns = mtime
		return 0
	}
	return info.Mtime_ns
}

func getTime(filename string) int64 {
	return _time(filename, 0)
}

func setTime(filename string, mtime int64) {
	_time(filename, mtime)
}

// Comments or comments out the line interpreter.
func comment(filename string, ok bool) {
	file, err := os.Open(filename, os.O_WRONLY, 0)
	if err != nil {
		goto _error
	}
	defer file.Close()

	if ok {
		if _, err = file.Write([]byte("//")); err != nil {
			goto _error
		}
	} else {
		if _, err = file.Write([]byte("#!")); err != nil {
			goto _error
		}
	}

	return

_error:
	fmt.Fprintf(os.Stderr, "Could not write: %s\n", err)
	os.Exit(EXIT_CODE)
}

// Executes a command and returns its exit code.
func run(cmd string, args []string, dir string) int {
	// Execute the command
	process, err := exec.Run(cmd, args, ENVIRON, dir,
		exec.PassThrough, exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not execute: \"%s\"\n",
			strings.Join(args, " "))
		os.Exit(EXIT_CODE)
	}

	// Wait for command completion
	message, err := process.Wait(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not wait for: \"%s\"\n",
			strings.Join(args, " "))
		os.Exit(EXIT_CODE)
	}

	return message.ExitStatus()
}

// Gets the toolchain.
func toolchain() (compiler, linker, archExt string) {
	arch_ext := map[string]string{
		"amd64": "6",
		"386":   "8",
		"arm":   "5",
	}

	// === Environment variables
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		goroot = os.Getenv("GOROOT_FINAL")
		if goroot == "" {
			fmt.Fprintf(os.Stderr, "Environment variable GOROOT neither"+
				" GOROOT_FINAL has been set\n")
			os.Exit(EXIT_CODE)
		}
	}

	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gobin = goroot + "/bin"
	}

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	// === Set toolchain
	archExt, ok := arch_ext[goarch]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown GOARCH: %s\n", goarch)
		os.Exit(EXIT_CODE)
	}

	compiler = path.Join(gobin, archExt+"g")
	linker = path.Join(gobin, archExt+"l")
	return
}


func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println(`Usage: goscript test.go`)
		return
	}

	sourceFile := args[1] // Relative path
	sourceDir, baseSourceFile := path.Split(sourceFile)
	// The executable is an hidden file.
	baseExecFile := "." + baseSourceFile[:len(baseSourceFile)-2] + ".gosc"
	execFile := path.Join(sourceDir, baseExecFile)

	// === Run the executable, if exist and it has not been modified
	if _, err := os.Stat(execFile); err == nil {
		sourceMtime := getTime(sourceFile)
		execMtime := getTime(execFile)

		if sourceMtime == execMtime {
			goto _run
		}
	}

	// === Check script extension
	if path.Ext(sourceFile) != ".go" {
		fmt.Fprintf(os.Stderr, "Wrong extension! It has to be \".go\"\n")
		os.Exit(EXIT_CODE)
	}

	// === Compile and link
	sourceMtime := getTime(sourceFile)
	//comment(sourceFile, true)
	compiler, linker, archExt := toolchain()

	ENVIRON = os.Environ()
	objectFile := "_go_." + archExt

	cmdArgs := []string{path.Base(compiler), "-o", objectFile, baseSourceFile}
	exitCode := run(compiler, cmdArgs, sourceDir)
	comment(sourceFile, false)
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	cmdArgs = []string{path.Base(linker), "-o", baseExecFile, objectFile}
	if exitCode = run(linker, cmdArgs, sourceDir); exitCode != 0 {
		os.Exit(exitCode)
	}

	// === Cleaning
	// Set mtime of executable just like the source file
	setTime(sourceFile, sourceMtime)
	setTime(execFile, sourceMtime)

	if err := os.Remove(path.Join(sourceDir, objectFile)); err != nil {
		fmt.Fprintf(os.Stderr, "Could not remove: %s\n", err)
		os.Exit(EXIT_CODE)
	}
	// ===

_run:
	exitCode = run(execFile, []string{baseExecFile}, "")
	os.Exit(exitCode)
}

