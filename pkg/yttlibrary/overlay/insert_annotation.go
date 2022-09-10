// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package overlay

import (
	"fmt"
	"github.com/k14s/starlark-go/starlark"
	"github.com/vmware-tanzu/carvel-ytt/pkg/template"
	tplcore "github.com/vmware-tanzu/carvel-ytt/pkg/template/core"
	"github.com/vmware-tanzu/carvel-ytt/pkg/yamltemplate"
)

const (
	InsertAnnotationKwargBefore string = "before"
	InsertAnnotationKwargAfter  string = "after"
	InsertAnnotationKwargVia    string = "via"
)

type InsertAnnotation struct {
	newItem template.EvaluationNode
	before  bool
	after   bool
	via     *starlark.Value
	thread  *starlark.Thread
}

func NewInsertAnnotation(newItem template.EvaluationNode, thread *starlark.Thread) (InsertAnnotation, error) {
	annotation := InsertAnnotation{
		newItem: newItem,
		thread:  thread,
	}
	anns := template.NewAnnotations(newItem)

	if !anns.Has(AnnotationInsert) {
		return annotation, fmt.Errorf(
			"Expected item to have '%s' annotation", AnnotationInsert)
	}

	kwargs := anns.Kwargs(AnnotationInsert)
	if len(kwargs) == 0 {
		return annotation, fmt.Errorf("Expected '%s' annotation to have "+
			"at least one keyword argument (before=..., after=...)", AnnotationInsert)
	}

	for _, kwarg := range kwargs {
		kwargName := string(kwarg[0].(starlark.String))

		switch kwargName {
		case "before":
			resultBool, err := tplcore.NewStarlarkValue(kwarg[1]).AsBool()
			if err != nil {
				return InsertAnnotation{}, err
			}
			annotation.before = resultBool

		case "after":
			resultBool, err := tplcore.NewStarlarkValue(kwarg[1]).AsBool()
			if err != nil {
				return InsertAnnotation{}, err
			}
			annotation.after = resultBool

		case "via":
			annotation.via = &kwarg[1]

		default:
			return annotation, fmt.Errorf(
				"Unknown '%s' annotation keyword argument '%s'", AnnotationInsert, kwargName)
		}
	}

	return annotation, nil
}

func (a InsertAnnotation) IsBefore() bool { return a.before }
func (a InsertAnnotation) IsAfter() bool  { return a.after }

func (a InsertAnnotation) Value(existingNode template.EvaluationNode) (interface{}, error) {
	newNode := a.newItem.DeepCopyAsInterface().(template.EvaluationNode)
	if a.via == nil {
		return newNode.GetValues()[0], nil
	}

	switch typedVal := (*a.via).(type) {
	case starlark.Callable:
		var existingVal interface{}
		if existingNode != nil {
			// Make sure original nodes are not affected in any way
			existingVal = existingNode.DeepCopyAsInterface().(template.EvaluationNode).GetValues()[0]
		} else {
			existingVal = nil
		}

		viaArgs := starlark.Tuple{
			yamltemplate.NewGoValueWithYAML(existingVal).AsStarlarkValue(),
		}

		// TODO check thread correctness
		result, err := starlark.Call(a.thread, *a.via, viaArgs, []starlark.Tuple{})
		if err != nil {
			return nil, err
		}

		return tplcore.NewStarlarkValue(result).AsGoValue()

	default:
		return nil, fmt.Errorf("Expected '%s' annotation keyword argument 'via'"+
			" to be function, but was %T", AnnotationInsert, typedVal)
	}
}
