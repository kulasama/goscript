package main

import (
	"exec"
	"fmt"
	"io"
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
				" GOROOT_FINAL is set\n")
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

// Copies the file `sourceFile` to a temporary directory.
func copyFile(dest, src string) {
	outFile, err := os.Open(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create: %s\n", err)
		os.Exit(EXIT_CODE)
	}
	defer outFile.Close()

	inFile, err := os.Open(src, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read: %s\n", err)
		os.Exit(EXIT_CODE)
	}

	// === Copy file
	_, err = io.WriteString(outFile, "//") // To comment the line interpreter.
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not write: %s\n", err)
		os.Exit(EXIT_CODE)
	}

	_, err = io.Copy(outFile, inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not copy: %s\n", err)
		os.Exit(EXIT_CODE)
	}
}

// Executes commands.
func run(cmd string, args []string, dir string) {
	// Execute the command
	process, err := exec.Run(cmd, args, ENVIRON, dir,
		exec.DevNull, exec.PassThrough, exec.PassThrough)
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

	// === Check script extension
	sourceFile := args[1] // Relative path
	baseSourceFile := path.Base(sourceFile)

	if path.Ext(sourceFile) != ".gos" {
		fmt.Fprintf(os.Stderr, "Wrong extension! It has to be \".gos\"\n")
		os.Exit(EXIT_CODE)
	}

	// === Copy to temporary file
	tempDir := os.TempDir()
	destFile := path.Join(tempDir, baseSourceFile)
	copyFile(destFile, sourceFile)

	// === Execute commands
	compiler, linker, archExt := toolchain()
	ENVIRON = os.Environ()

	cmdArgs := []string{path.Base(compiler), baseSourceFile}
	run(compiler, cmdArgs, tempDir)

	// Get extensions
	objectFile := baseSourceFile[:len(baseSourceFile)-4] + "." + archExt
	objectFile = path.Join(tempDir, objectFile)
	outputFile := sourceFile[:len(sourceFile)-4] + ".goc"

	cmdArgs = []string{path.Base(linker), "-o", outputFile, objectFile}
	run(linker, cmdArgs, "")

	// === Cleaning
	for _, file := range []string{destFile, objectFile} {
		if err := os.Remove(file); err != nil {
			fmt.Fprintf(os.Stderr, "Could not remove: %s\n", err)
			os.Exit(EXIT_CODE)
		}
	}

	println("OK")
//	os.Exit(0)
}

