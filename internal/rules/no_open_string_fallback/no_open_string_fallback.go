package no_open_string_fallback

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// inlineString returns the string literal an expression is, under any
// parentheses and type assertions, or nil. An empty string is left to the
// plugin's syntactic rule, which reports it whatever the type.
func inlineString(expression *ast.Expression) *ast.Node {
	if expression == nil {
		return nil
	}
	value := ast.SkipParentheses(expression)
	for ast.IsAsExpression(value) || ast.IsSatisfiesExpression(value) || ast.IsParenthesizedExpression(value) {
		value = ast.SkipParentheses(value.Expression())
	}
	if !ast.IsStringLiteral(value) && !ast.IsNoSubstitutionTemplateLiteral(value) {
		return nil
	}
	if value.Text() == "" {
		return nil
	}
	return value
}

// invents reports whether a literal fills a type that does not name it: an
// open string, or a literal union the value is outside. A type the checker
// cannot resolve here, such as any, unknown or a type parameter, is left alone.
func invents(ctx rule.RuleContext, filled *checker.Type, literal string) bool {
	unresolved := checker.TypeFlagsAnyOrUnknown | checker.TypeFlagsNever | checker.TypeFlagsTypeParameter
	open := false
	member := false
	literalUnion := false
	for _, part := range utils.UnionTypeParts(utils.GetNonNullableType(ctx.TypeChecker, filled)) {
		if utils.IsTypeFlagSet(part, unresolved) {
			return false
		}
		if utils.IsTypeFlagSet(part, checker.TypeFlagsString) {
			open = true
		}
		if utils.IsTypeFlagSet(part, checker.TypeFlagsStringLiteral) {
			literalUnion = true
			if value, ok := part.AsLiteralType().Value().(string); ok && value == literal {
				member = true
			}
		}
	}
	return open || (literalUnion && !member)
}

// inParameter reports whether a binding element sits in a parameter's
// pattern, where its default is the function's documented choice.
func inParameter(node *ast.Node) bool {
	for at := node.Parent; at != nil; at = at.Parent {
		switch at.Kind {
		case ast.KindBindingElement, ast.KindObjectBindingPattern, ast.KindArrayBindingPattern:
			continue
		case ast.KindParameter:
			return true
		default:
			return false
		}
	}
	return false
}

func nodeText(sourceFile *ast.SourceFile, node *ast.Node) string {
	range_ := utils.TrimNodeTextRange(sourceFile, node)
	return sourceFile.Text()[range_.Pos():range_.End()]
}

const advice = "usually just fails later, in validation. Make the value required, or handle its absence here."

var NoOpenStringFallbackRule = rule.Rule{
	Name: "no-open-string-fallback",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindBinaryExpression: func(node *ast.Node) {
				expression := node.AsBinaryExpression()
				operator := expression.OperatorToken.Kind
				if operator != ast.KindQuestionQuestionToken && operator != ast.KindQuestionQuestionEqualsToken {
					return
				}
				literal := inlineString(expression.Right)
				if literal == nil {
					return
				}
				if !invents(ctx, ctx.TypeChecker.GetTypeAtLocation(expression.Left), literal.Text()) {
					return
				}
				ctx.ReportNode(node, rule.RuleMessage{
					Id:          "placeholder",
					Description: fmt.Sprintf("A `%s` placeholder for a missing value %s", nodeText(ctx.SourceFile, literal), advice),
				})
			},
			ast.KindBindingElement: func(node *ast.Node) {
				initializer := node.Initializer()
				name := node.Name()
				if initializer == nil || name == nil || !ast.IsIdentifier(name) || inParameter(node) {
					return
				}
				literal := inlineString(initializer)
				if literal == nil {
					return
				}
				// The binding's type is the declared one without undefined, so a
				// union default that is a member resolves to the union itself
				if !invents(ctx, ctx.TypeChecker.GetTypeAtLocation(name), literal.Text()) {
					return
				}
				ctx.ReportNode(node, rule.RuleMessage{
					Id:          "placeholder",
					Description: fmt.Sprintf("A `= %s` placeholder for a missing `%s` %s", nodeText(ctx.SourceFile, literal), name.Text(), advice),
				})
			},
		}
	},
}
