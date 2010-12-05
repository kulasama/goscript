// TODO: The executable file will be hidden in Windows when Go can access to
// this Win32 API:
// http://msdn.microsoft.com/en-us/library/aa365535%28VS.85%29.aspx

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

// Comment or comment out the interpreter line.
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

// Executes commands.
func run(cmd string, args []string, dir string) {
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

	// Exit if it has failed
	if exitCode := message.ExitStatus(); exitCode != 0 {
		os.Exit(exitCode)
	}
}


func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println(`Usage: Insert "#!/usr/bin/env goscript" in your Go script`)
		return
	}

	sourceFile := args[1] // Relative path
	sourceDir, baseSourceFile := path.Split(sourceFile)
	// The executable is an hidden file.
	baseExecFile := "." + baseSourceFile[:len(baseSourceFile)-2] + ".goc"
	execFile := path.Join(sourceDir, baseExecFile)

	// === Run the executable, if exist
	if _, err := os.Stat(execFile); err == nil {
		run(execFile, []string{baseExecFile}, "")
		os.Exit(0)
	}

	// === Check script extension
	if path.Ext(sourceFile) != ".g" {
		fmt.Fprintf(os.Stderr, "Wrong extension! It has to be \".g\"\n")
		os.Exit(EXIT_CODE)
	}

	// === Compile and link
	comment(sourceFile, true)
	compiler, linker, archExt := toolchain()

	ENVIRON = os.Environ()
	objectFile := "_go_." + archExt

	cmdArgs := []string{path.Base(compiler), "-o", objectFile, baseSourceFile}
	run(compiler, cmdArgs, sourceDir)

	cmdArgs = []string{path.Base(linker), "-o", baseExecFile, objectFile}
	run(linker, cmdArgs, sourceDir)

	// === Cleaning
	comment(sourceFile, false)

	if err := os.Remove(path.Join(sourceDir, objectFile)); err != nil {
		fmt.Fprintf(os.Stderr, "Could not remove: %s\n", err)
		os.Exit(EXIT_CODE)
	}
	// ===

	run(execFile, []string{baseExecFile}, "")
}

