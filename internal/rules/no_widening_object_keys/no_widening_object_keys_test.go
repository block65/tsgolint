package no_widening_object_keys

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/bundled"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/microsoft/typescript-go/shim/vfs/cachedvfs"
	"github.com/microsoft/typescript-go/shim/vfs/osvfs"
	"github.com/typescript-eslint/tsgolint/internal/diagnostic"
	"github.com/typescript-eslint/tsgolint/internal/linter"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

//go:embed plugin_cases.json
var pluginCases []byte

// pluginCaseBaseFS backs runPluginCaseDiagnostics with the same bundled
// filesystem rule_tester.RunRuleTester uses, so the two runs see identical
// lib files.
var pluginCaseBaseFS = cachedvfs.From(bundled.WrapFS(osvfs.FS()))

// tsconfigOverride reports the TSConfig override to pass to rule_tester, and
// to runPluginCaseDiagnostics, for a case. A case that ships its own
// "tsconfig.json" content under "files" is compiled against that file
// instead of the package's default tsconfig.minimal.json.
func tsconfigOverride(files map[string]string) string {
	if _, ok := files["tsconfig.json"]; ok {
		return "tsconfig.json"
	}
	return ""
}

// runPluginCaseDiagnostics runs r directly against code (mirroring
// rule_tester.RunRuleTester's own internal run) so the bridge can inspect
// diagnostic message text. rule_tester has no public API for that, and it
// must not be modified to add one, so this is a self-contained copy of its
// linting setup restricted to what message assertions need.
func runPluginCaseDiagnostics(t *testing.T, r *rule.Rule, code string, filename string, options any, tsconfigPathOverride string, extraFiles map[string]string) []rule.RuleDiagnostic {
	t.Helper()

	rootDir := fixtures.GetRootDir()
	tsconfigPath := "tsconfig.minimal.json"
	if tsconfigPathOverride != "" {
		tsconfigPath = tsconfigPathOverride
	}

	if filename == "" {
		filename = "file.ts"
	}

	resolvedFileName := tspath.ResolvePath(rootDir, filename)
	virtualFiles := map[string]string{resolvedFileName: code}
	for relativePath, source := range extraFiles {
		virtualFiles[tspath.ResolvePath(rootDir, relativePath)] = source
	}
	fs := utils.NewOverlayVFS(pluginCaseBaseFS, virtualFiles)
	host := utils.CreateCompilerHost(rootDir, fs)

	program, internalDiagnostics, err := utils.CreateProgram(true, fs, rootDir, tsconfigPath, host, false)
	if err != nil {
		t.Fatalf("couldn't create program. code:\n%v\nerror: %v", code, err)
	}
	if len(internalDiagnostics) > 0 {
		t.Fatalf("couldn't create program due to internal diagnostics: %+v", internalDiagnostics)
	}

	sourceFile := program.GetSourceFile(filename)
	if sourceFile == nil {
		sourceFile = program.GetSourceFile(resolvedFileName)
	}
	if sourceFile == nil {
		t.Fatalf("couldn't get source file: %v (resolved: %v)", filename, resolvedFileName)
	}

	var diagnosticsMu sync.Mutex
	diagnostics := make([]rule.RuleDiagnostic, 0, 1)

	err = linter.RunLinterOnProgram(linter.RunLinterOnProgramOptions{
		LogLevel: utils.LogLevelNormal,
		Program:  program,
		Files:    []*ast.SourceFile{sourceFile},
		Workers:  1,
		GetRulesForFile: func(sourceFile *ast.SourceFile) []linter.ConfiguredRule {
			return []linter.ConfiguredRule{
				{
					Name: "test",
					Run: func(ctx rule.RuleContext) rule.RuleListeners {
						return r.Run(ctx, options)
					},
				},
			}
		},
		OnDiagnostic: func(d rule.RuleDiagnostic) {
			diagnosticsMu.Lock()
			defer diagnosticsMu.Unlock()
			diagnostics = append(diagnostics, d)
		},
		OnInternalDiagnostic: func(d diagnostic.Internal) {},
		Fixes: linter.Fixes{
			Fix:            true,
			FixSuggestions: true,
		},
		TypeErrors: linter.TypeErrors{
			ReportSyntactic: false,
			ReportSemantic:  false,
		},
	})
	if err != nil {
		t.Fatalf("error running linter. code:\n%v\nerror: %v", code, err)
	}

	return diagnostics
}

// assertPluginCaseMessages checks that the diagnostics produced for an
// invalid case carry the expected/forbidden message substrings from the
// original plugin's test case, without weakening the message id/line
// assertions rule_tester.RunRuleTester already performs.
func assertPluginCaseMessages(t *testing.T, r *rule.Rule, code, filename string, options any, tsconfigPathOverride string, extraFiles map[string]string, want, notWant []string) {
	t.Helper()
	if len(want) == 0 && len(notWant) == 0 {
		return
	}

	diagnostics := runPluginCaseDiagnostics(t, r, code, filename, options, tsconfigPathOverride, extraFiles)

	descriptions := make([]string, len(diagnostics))
	for i, d := range diagnostics {
		descriptions[i] = d.Message.Description
	}
	joined := strings.Join(descriptions, "\n")

	for _, expected := range want {
		if !strings.Contains(joined, expected) {
			t.Errorf("expected a diagnostic message to contain %q, got: %v", expected, descriptions)
		}
	}
	for _, forbidden := range notWant {
		if strings.Contains(joined, forbidden) {
			t.Errorf("expected no diagnostic message to contain %q, got: %v", forbidden, descriptions)
		}
	}
}

func TestPluginCases(t *testing.T) {
	var cases []struct {
		Name        string            `json:"name"`
		Code        string            `json:"code"`
		Filename    string            `json:"filename"`
		Files       map[string]string `json:"files"`
		Options     any               `json:"options"`
		Expect      []string          `json:"expect"`
		Messages    []string          `json:"messages"`
		NotMessages []string          `json:"notMessages"`
	}
	if err := json.Unmarshal(pluginCases, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			code := strings.TrimPrefix(test.Code, "\n")
			tsconfig := tsconfigOverride(test.Files)
			if len(test.Expect) == 0 {
				rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.minimal.json", t, &NoWideningObjectKeysRule,
					[]rule_tester.ValidTestCase{{Code: code, FileName: test.Filename, Files: test.Files, Options: test.Options, TSConfig: tsconfig}}, nil)
			} else {
				errors := make([]rule_tester.InvalidTestCaseError, 0, len(test.Expect))
				for _, expected := range test.Expect {
					_, line, ok := strings.Cut(expected, "@")
					if !ok {
						t.Fatal(expected)
					}
					number, err := strconv.Atoi(line)
					if err != nil {
						t.Fatal(err)
					}
					errors = append(errors, rule_tester.InvalidTestCaseError{MessageId: "widenedKeys", Line: number})
				}
				rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.minimal.json", t, &NoWideningObjectKeysRule, nil,
					[]rule_tester.InvalidTestCase{{Code: code, FileName: test.Filename, Files: test.Files, Options: test.Options, Errors: errors, TSConfig: tsconfig}})

				assertPluginCaseMessages(t, &NoWideningObjectKeysRule, code, test.Filename, test.Options, tsconfig, test.Files, test.Messages, test.NotMessages)
			}
		})
	}
}
