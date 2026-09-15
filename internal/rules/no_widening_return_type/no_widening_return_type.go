package no_widening_return_type

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// A function with a return annotation and a body. Constructors and accessors
// are intentionally omitted to match the source rule's scope.
func isAnnotatedFunction(node *ast.Node) bool {
	if node == nil || node.Body() == nil || node.Type() == nil {
		return false
	}

	switch {
	case ast.IsFunctionDeclaration(node), ast.IsMethodDeclaration(node),
		ast.IsArrowFunction(node), ast.IsFunctionExpression(node):
		return ast.GetFunctionFlags(node)&ast.FunctionFlagsGenerator == 0
	default:
		return false
	}
}

// returnedTypes collects the types a function body hands back. The native AST
// helper deliberately does not descend into nested functions, so an inner
// function keeps its own return contract.
func returnedTypes(typeChecker *checker.Checker, functionNode *ast.Node) []*checker.Type {
	body := functionNode.Body()
	if body == nil {
		return nil
	}

	if !ast.IsBlock(body) {
		return []*checker.Type{typeChecker.GetTypeAtLocation(body)}
	}

	var types []*checker.Type
	ast.ForEachReturnStatement(body, func(statement *ast.Node) bool {
		if expression := statement.Expression(); expression != nil {
			types = append(types, typeChecker.GetTypeAtLocation(expression))
		} else {
			types = append(types, checker.Checker_undefinedType(typeChecker))
		}
		return false
	})

	// TypeScript's checker has already built the control-flow graph. Its
	// HasImplicitReturn flag is the closest public native signal available from
	// the shim: a reachable function end contributes undefined. The binder can
	// conservatively retain this flag for some exhaustive switch forms, so this
	// remains intentionally conservative until a checker-native flow query is
	// exposed.
	if len(types) != 0 && functionNode.Flags&ast.NodeFlagsHasImplicitReturn != 0 {
		types = append(types, checker.Checker_undefinedType(typeChecker))
	}

	return types
}

func answersAContract(annotation *ast.Node, returned []*checker.Type) bool {
	// A named annotation communicates an intentional contract. A type literal
	// remains an inference boundary and should still be reported.
	if !ast.IsTypeReferenceNode(annotation) {
		return false
	}

	// Each element is the type of one return statement's expression, taken as
	// the checker resolved it there - never decomposed. A statement whose own
	// expression is a union (a ternary between two shapes, say) is not itself
	// an object type, so it fails the check below exactly as it would if the
	// source rule inspected it directly, rather than being split into parts
	// that might individually pass.
	objectFlags := checker.ObjectFlagsAnonymous | checker.ObjectFlagsMapped
	for _, part := range returned {
		if !utils.IsTypeFlagSet(part, checker.TypeFlagsObject) || checker.Type_objectFlags(part)&objectFlags == 0 {
			return false
		}
	}
	return true
}

func unresolved(t *checker.Type) bool {
	return utils.IsTypeFlagSet(t, checker.TypeFlagsAny|checker.TypeFlagsNever)
}

var NoWideningReturnTypeRule = rule.Rule{
	Name: "no-widening-return-type",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoWideningReturnTypeOptions](options, "no-widening-return-type")

		check := func(functionNode *ast.Node) {
			annotation := functionNode.Type()
			if annotation == nil || ast.IsTypePredicateNode(annotation) {
				return
			}

			functionFlags := ast.GetFunctionFlags(functionNode)
			unwrap := func(t *checker.Type) *checker.Type {
				if functionFlags&ast.FunctionFlagsAsync == 0 {
					return t
				}
				if awaited := checker.Checker_getAwaitedType(ctx.TypeChecker, t); awaited != nil {
					return awaited
				}
				return t
			}

			returned := returnedTypes(ctx.TypeChecker, functionNode)
			if len(returned) == 0 {
				return
			}
			for index := range returned {
				returned[index] = unwrap(returned[index])
				if returned[index] == nil || unresolved(returned[index]) {
					return
				}
			}

			declared := unwrap(checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, annotation))
			if declared == nil || unresolved(declared) {
				return
			}
			actual := checker.Checker_getUnionType(ctx.TypeChecker, returned)
			assignable := func(from, to *checker.Type) bool {
				return checker.Checker_isTypeAssignableTo(ctx.TypeChecker, from, to)
			}

			// Incompatible annotations and equivalent types are handled by the
			// compiler or are not widening, respectively.
			if !assignable(actual, declared) || assignable(declared, actual) {
				return
			}
			if opts.Contracts && answersAContract(annotation, returned) {
				return
			}

			ctx.ReportNode(annotation, rule.RuleMessage{
				Id:          "widenedReturn",
				Description: fmt.Sprintf("Return type `%s` is wider than the `%s` the body returns, so every caller loses that. Annotate the narrower type, or drop the annotation and let it infer.", ctx.TypeChecker.TypeToString(declared), ctx.TypeChecker.TypeToString(actual)),
			})
		}

		visit := func(node *ast.Node) {
			if isAnnotatedFunction(node) {
				check(node)
			}
		}
		return rule.RuleListeners{
			ast.KindFunctionDeclaration: visit,
			ast.KindMethodDeclaration:   visit,
			ast.KindArrowFunction:       visit,
			ast.KindFunctionExpression:  visit,
		}
	},
}
