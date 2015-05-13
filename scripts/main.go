// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File main.go is intended to be used with go generate.
// It reads the contents of any .lua file ins the scripts
// directory, then it generates a go source file called
// scritps.go which converts the file contents to a string
// and assigns each script to a variable so they can be invoked.

package main

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	// scriptsPath is the path of the directory which holds lua scripts.
	scriptsPath string
	// destPath is the path to a file where the generated go code will be written.
	destPath string
	// tmplPath is the path to a .tmpl file which will be used to generate go code.
	tmplPath string
)

// script is a representation of a lua script file.
type script struct {
	// VarName is the variable name that the script will be assigned to in the generated go code.
	VarName string
	// Src is the contents of the original .lua file.
	Src string
}

func init() {
	// Use build to find the directory where this file lives. This always works as
	// long as you have go installed, even if you have multiple GOPATHs or are using
	// dependency management tools.
	pkg, err := build.Import("github.com/albrow/zoom", "", build.FindOnly)
	if err != nil {
		panic(err)
	}
	// Configure the required paths
	scriptsPath = filepath.Join(pkg.Dir, "scripts")
	destPath = filepath.Clean(filepath.Join(scriptsPath, "..", "scripts.go"))
	tmplPath = filepath.Join(scriptsPath, "scripts.go.tmpl")
}

func main() {
	scripts, err := findScripts(scriptsPath)
	if err != nil {
		panic(err)
	}
	if err := generateFile(scripts, tmplPath, destPath); err != nil {
		panic(err)
	}
}

// findScripts finds all the .lua script files in the given path
// and creates a script object for each one. It returns a slice of
// scripts or an error if there was a problem reading any of the files.
func findScripts(path string) ([]script, error) {
	filenames, err := filepath.Glob(filepath.Join(path, "*.lua"))
	if err != nil {
		return nil, err
	}
	scripts := []script{}
	for _, filename := range filenames {
		script := script{
			VarName: convertUnderscoresToCamelCase(strings.TrimSuffix(filepath.Base(filename), ".lua")) + "Script",
		}
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		script.Src = string(src)
		scripts = append(scripts, script)
	}
	return scripts, nil
}

// convertUnderscoresToCamelCase converts a string of the form
// foo_bar_baz to fooBarBaz.
func convertUnderscoresToCamelCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	result := ""
	shouldUpper := false
	for _, char := range s {
		if char == '_' {
			shouldUpper = true
			continue
		}
		if shouldUpper {
			result += strings.ToUpper(string(char))
		} else {
			result += string(char)
		}
		shouldUpper = false
	}
	return result
}

// generateFile generates go source code and writes to
// the source file located at dest (creating it if needed).
// It executes the template located at tmplFile with scripts
// as the context.
func generateFile(scripts []script, tmplFile string, dest string) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return err
	}
	return tmpl.Execute(destFile, scripts)
}
