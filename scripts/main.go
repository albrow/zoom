package main

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var scriptsPath string

type script struct {
	VarName  string
	Filename string
	Src      string
}

func init() {
	pkg, err := build.Import("github.com/albrow/zoom", "", build.FindOnly)
	if err != nil {
		panic(err)
	}
	scriptsPath = filepath.Join(pkg.Dir, "scripts")
}

func main() {
	scripts, err := findScripts(scriptsPath)
	if err != nil {
		panic(err)
	}
	if err := generateFile(scripts); err != nil {
		panic(err)
	}
}

func findScripts(path string) ([]script, error) {
	filenames, err := filepath.Glob(filepath.Join(path, "*.lua"))
	if err != nil {
		return nil, err
	}
	scripts := []script{}
	for _, filename := range filenames {
		script := script{
			VarName:  convertUnderscoresToCamelCase(strings.TrimSuffix(filepath.Base(filename), ".lua")) + "Script",
			Filename: filename,
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

func generateFile(scripts []script) error {
	destName := filepath.Clean(filepath.Join(scriptsPath, "..", "scripts.go"))
	destFile, err := os.Create(destName)
	if err != nil {
		return err
	}
	tmpl, err := template.ParseFiles(filepath.Join(scriptsPath, "scripts.go.tmpl"))
	if err != nil {
		return err
	}
	return tmpl.Execute(destFile, scripts)
}
