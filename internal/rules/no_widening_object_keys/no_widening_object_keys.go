package no_widening_object_keys

import (
	"fmt"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func callee(node *ast.Node) (*ast.Node, string) {
	expression := node.AsCallExpression().Expression
	if ast.IsPropertyAccessExpression(expression) {
		return expression.AsPropertyAccessExpression().Expression, expression.Name().Text()
	}
	if ast.IsElementAccessExpression(expression) {
		member := expression.AsElementAccessExpression()
		if ast.IsStringLiteral(member.ArgumentExpression) {
			return member.Expression, member.ArgumentExpression.Text()
		}
	}
	return nil, ""
}

// A destructuring pattern or a computed method name is never the helper
func declaredName(node *ast.Node) string {
	if !ast.IsFunctionDeclaration(node) && !ast.IsMethodDeclaration(node) && !ast.IsVariableDeclaration(node) {
		return ""
	}
	name := node.Name()
	if name == nil || !ast.IsIdentifier(name) {
		return ""
	}
	return name.Text()
}

func inHelper(node *ast.Node, name string) bool {
	for at := node.Parent; at != nil; at = at.Parent {
		if declaredName(at) == name {
			return true
		}
	}
	return false
}

func finiteKeys(ctx rule.RuleContext, t *checker.Type) bool {
	for _, part := range utils.UnionTypeParts(t) {
		if utils.IsTypeFlagSet(part, checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever|checker.TypeFlagsTypeParameter) || len(checker.Checker_getIndexInfosOfType(ctx.TypeChecker, part)) > 0 || len(checker.Checker_getPropertiesOfType(ctx.TypeChecker, part)) == 0 {
			return false
		}
	}
	return true
}

var NoWideningObjectKeysRule = rule.Rule{
	Name: "no-widening-object-keys",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoWideningObjectKeysOptions](options, "no-widening-object-keys")
		helpers := map[string]string{"keys": opts.Keys, "entries": opts.Entries}
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				target, method := callee(node)
				helper, supported := helpers[method]
				call := node.AsCallExpression()
				if !supported || target == nil || !ast.IsIdentifier(target) || target.Text() != "Object" || call.Arguments == nil || len(call.Arguments.Nodes) != 1 || inHelper(node, helper) {
					return
				}
				if parent := node.Parent; ast.IsPropertyAccessExpression(parent) && parent.Name().Text() == "length" {
					return
				}
				if symbol := ctx.TypeChecker.GetSymbolAtLocation(target); symbol != nil {
					for _, decl := range symbol.Declarations {
						if ast.GetSourceFileOfNode(decl) == ctx.SourceFile {
							return
						}
					}
				}
				t := ctx.TypeChecker.GetTypeAtLocation(call.Arguments.Nodes[0])
				if !finiteKeys(ctx, t) {
					return
				}
				ctx.ReportNode(node, rule.RuleMessage{
					Id:          "widenedKeys",
					Description: fmt.Sprintf("`Object.%s` is typed as if every key were a `string`, so the `%s` keys are lost. Call `%s`, which infers them from what it is given. One cast stays inside that helper, so it concentrates the unsoundness rather than removing it.", method, ctx.TypeChecker.TypeToString(t), helper),
				})
			},
		}
	},
}
