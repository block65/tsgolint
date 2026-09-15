package no_widening_alias

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// aliasesValue identifies the expressions for which an annotation can hide
// information that was inferred at the value's declaration. Calls and fresh
// literals are contextually typed, so they are deliberately excluded.
func aliasesValue(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	value := ast.SkipParentheses(expression)
	return ast.IsIdentifier(value) || ast.IsPropertyAccessExpression(value) || ast.IsElementAccessExpression(value)
}

func typeConstituents(t *checker.Type) []*checker.Type {
	return utils.UnionTypeParts(t)
}

// addsKeys detects the key widening case separately from ordinary value type
// widening. A target may be mutually assignable with the source while still
// accepting keys that the source table does not declare (for example a
// Partial<Record<"known" | "extra", ...>> target).
func addsKeys(ctx rule.RuleContext, actual *checker.Type, declared *checker.Type) bool {
	original := make(map[string]struct{})
	var indexes []*checker.IndexInfo
	for _, typ := range typeConstituents(actual) {
		for _, property := range checker.Checker_getPropertiesOfType(ctx.TypeChecker, typ) {
			if property != nil {
				original[property.Name] = struct{}{}
			}
		}
		indexes = append(indexes, checker.Checker_getIndexInfosOfType(ctx.TypeChecker, typ)...)
	}

	for _, typ := range typeConstituents(declared) {
		extraProperty := false
		for _, property := range checker.Checker_getPropertiesOfType(ctx.TypeChecker, typ) {
			if property == nil {
				continue
			}
			if _, exists := original[property.Name]; exists {
				continue
			}
			key := checker.Checker_getStringLiteralType(ctx.TypeChecker, property.Name)
			covered := false
			for _, index := range indexes {
				if index != nil && checker.Checker_isTypeAssignableTo(ctx.TypeChecker, key, index.KeyType()) {
					covered = true
					break
				}
			}
			if !covered {
				extraProperty = true
				break
			}
		}

		extraIndex := false
		for _, index := range checker.Checker_getIndexInfosOfType(ctx.TypeChecker, typ) {
			if index == nil {
				continue
			}
			covered := false
			for _, originalIndex := range indexes {
				if originalIndex != nil && checker.Checker_isTypeAssignableTo(ctx.TypeChecker, index.KeyType(), originalIndex.KeyType()) {
					covered = true
					break
				}
			}
			if !covered {
				extraIndex = true
				break
			}
		}

		if extraProperty || extraIndex {
			return true
		}
	}
	return false
}

func nodeText(sourceFile *ast.SourceFile, node *ast.Node) string {
	range_ := utils.TrimNodeTextRange(sourceFile, node)
	return sourceFile.Text()[range_.Pos():range_.End()]
}

var NoWideningAliasRule = rule.Rule{
	Name: "no-widening-alias",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindVariableDeclaration: func(node *ast.Node) {
				if ctx.SourceFile.IsDeclarationFile {
					return
				}
				declaration := node.AsVariableDeclaration()
				if declaration.Type == nil || declaration.Initializer == nil || !ast.IsIdentifier(declaration.Name()) || !aliasesValue(declaration.Initializer) {
					return
				}
				if !ast.IsVariableDeclarationList(node.Parent) || node.Parent.Flags&ast.NodeFlagsConst == 0 {
					return
				}

				actual := ctx.TypeChecker.GetTypeAtLocation(declaration.Initializer)
				declared := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, declaration.Type)
				unresolved := checker.TypeFlagsAny | checker.TypeFlagsNever | checker.TypeFlagsTypeParameter
				for _, typ := range append(typeConstituents(actual), typeConstituents(declared)...) {
					if utils.IsTypeFlagSet(typ, unresolved) {
						return
					}
				}
				if !checker.Checker_isTypeAssignableTo(ctx.TypeChecker, actual, declared) {
					return
				}

				widerKeys := addsKeys(ctx, actual, declared)
				if !widerKeys && checker.Checker_isTypeAssignableTo(ctx.TypeChecker, declared, actual) {
					return
				}

				keys := ""
				if widerKeys {
					keys = " It accepts keys the original type does not declare, so narrow the key where it comes in, with a guard or a parse, rather than widening the table."
				}
				ctx.ReportNode(declaration.Name(), rule.RuleMessage{
					Id:          "widenedAlias",
					Description: fmt.Sprintf("Annotation `%s` widens `%s` from `%s`. Drop the annotation to keep the inferred type.%s", nodeText(ctx.SourceFile, declaration.Type), nodeText(ctx.SourceFile, declaration.Initializer), ctx.TypeChecker.TypeToString(actual), keys),
				})
			},
		}
	},
}
