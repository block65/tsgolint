package define_messages_keys

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func intlImport(node *ast.Node) bool {
	if node == nil || !ast.IsImportDeclaration(node) {
		return false
	}
	source := node.AsImportDeclaration().ModuleSpecifier
	return ast.IsStringLiteral(source) && (source.Text() == "react-intl" || source.Text() == "@formatjs/intl")
}

func definesMessages(ctx rule.RuleContext, expression *ast.Node) bool {
	var target *ast.Node
	namespace := false
	if ast.IsIdentifier(expression) {
		target = expression
	} else if ast.IsPropertyAccessExpression(expression) && expression.Name().Text() == "defineMessages" {
		target = expression.AsPropertyAccessExpression().Expression
		namespace = true
	} else {
		return false
	}
	symbol := ctx.TypeChecker.GetSymbolAtLocation(target)
	if symbol == nil {
		return false
	}
	for _, declaration := range symbol.Declarations {
		if namespace {
			if ast.IsNamespaceImport(declaration) && intlImport(declaration.Parent.Parent) {
				return true
			}
		} else if ast.IsImportSpecifier(declaration) {
			specifier := declaration.AsImportSpecifier()
			imported := specifier.PropertyName
			if imported == nil {
				imported = declaration.Name()
			}
			if imported.Text() == "defineMessages" && intlImport(declaration.Parent.Parent.Parent) {
				return true
			}
		}
	}
	return false
}

func literalKeys(t *checker.Type) bool {
	parts := utils.UnionTypeParts(t)
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !utils.IsTypeFlagSet(part, checker.TypeFlagsStringLiteral|checker.TypeFlagsNumberLiteral) {
			return false
		}
	}
	return true
}

// inferredKeys reports whether the call resolved to an overload with a
// `const` type parameter, such as react-intl 12's
// `defineMessages<const D extends Record<string, MessageDescriptor>>`, and
// passes an object whose keys are all known. Such a call keeps the literal
// keys without a type argument, and adding one would select a legacy overload.
func inferredKeys(ctx rule.RuleContext, node *ast.Node) bool {
	call := node.AsCallExpression()
	if call.Arguments == nil || len(call.Arguments.Nodes) == 0 {
		return false
	}
	signature := checker.Checker_getResolvedSignature(ctx.TypeChecker, node, nil, checker.CheckModeNormal)
	if signature == nil {
		return false
	}
	declaration := checker.Signature_declaration(signature)
	if declaration == nil {
		return false
	}
	constGeneric := false
	for _, parameter := range declaration.TypeParameters() {
		if ast.HasSyntacticModifier(parameter, ast.ModifierFlagsConst) {
			constGeneric = true
			break
		}
	}
	if !constGeneric {
		return false
	}
	argument := ctx.TypeChecker.GetTypeAtLocation(call.Arguments.Nodes[0])
	return argument != nil &&
		utils.IsTypeFlagSet(argument, checker.TypeFlagsObject) &&
		len(checker.Checker_getIndexInfosOfType(ctx.TypeChecker, argument)) == 0
}

var DefineMessagesKeysRule = rule.Rule{
	Name: "define-messages-keys",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if !definesMessages(ctx, call.Expression) {
					return
				}
				var key *ast.Node
				if call.TypeArguments != nil && len(call.TypeArguments.Nodes) > 0 {
					key = call.TypeArguments.Nodes[0]
				}
				if key != nil && literalKeys(checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, key)) {
					return
				}
				if key == nil {
					if inferredKeys(ctx, node) {
						return
					}
					key = node
				}
				ctx.ReportNode(key, rule.RuleMessage{
					Id:          "finiteKeys",
					Description: "Give `defineMessages` an explicit finite union of string or number literals, such as `defineMessages<MessageKey>(...)`, or pass an object literal to an overload that keeps its keys with a `const` type parameter. Broad, unresolved, and unconstrained generic types are not message keys.",
				})
			},
		}
	},
}
