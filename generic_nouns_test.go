package espyna

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var genericIdentifierNoun = regexp.MustCompile(`ReportCard|reportCard|Teacher|teacher[A-Z]|Student|student[A-Z]|Enrollment|enrollment[A-Z]|Homeroom|homeroom[A-Z]|Deportment|deportment[A-Z]`)
var genericStringNoun = regexp.MustCompile(`(?i)report[-_ ]?cards?|teacher|student|enrol|homeroom|deportment`)
var persistedNounLiterals = map[string]bool{
	"report_card": true, "templates/report_card": true,
	"student_name": true, "page_student_name": true,
	"teacher_line": true, "section_name": true, "students": true,
}
var genericVerbWording = regexp.MustCompile(`(?i)enroll members|members enrolled|no members enrolled|enrolled in payroll|schedule enrollment|will be enrolled from active seats`)

func TestNoVerticalNounIdentifiers(t *testing.T) {
	root := "."
	fset := token.NewFileSet()
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "vendor" || entry.Name() == "node_modules" || entry.Name() == "pkg" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.Ident:
				if genericIdentifierNoun.MatchString(n.Name) {
					violations = append(violations, fmt.Sprintf("%s:%d: identifier %s", path, fset.Position(n.Pos()).Line, n.Name))
				}
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					break
				}
				value, err := strconv.Unquote(n.Value)
				if err != nil || !genericStringNoun.MatchString(value) {
					break
				}
				if persistedNounLiterals[value] || genericVerbWording.MatchString(value) {
					break
				}
				violations = append(violations, fmt.Sprintf("%s:%d: string contains %q", path, fset.Position(n.Pos()).Line, genericStringNoun.FindString(value)))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("vertical nouns in generic code:\n%s", strings.Join(violations, "\n"))
	}
}
