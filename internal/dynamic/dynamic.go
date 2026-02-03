// Package dynamic provides utilities for converting between Terraform's dynamic types
// and Go's native types. It handles bidirectional conversion between types.Dynamic
// values and standard Go types like strings, numbers, bools, maps, and slices.
package dynamic

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FromDynamic converts a Terraform Dynamic value to a native Go type.
// It handles null values, primitives (string, bool, number), and complex types (list, map, object).
//
// Parameters:
//   - dynVal: The Terraform Dynamic value to convert
//
// Returns a Go value of the appropriate type:
//   - nil for null values
//   - string for String values
//   - bool for Bool values
//   - float64 for Number values
//   - []any for List values
//   - map[string]any for Map and Object values
//   - string representation for unknown types
func FromDynamic(dynVal types.Dynamic) any {
	if dynVal.IsNull() || dynVal.IsUnderlyingValueNull() {
		return nil
	}

	underlyingValue := dynVal.UnderlyingValue()

	switch v := underlyingValue.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		bigFloat := v.ValueBigFloat()
		if bigFloat != nil {
			f64, _ := bigFloat.Float64()
			return f64
		}
		return nil
	case types.List:
		elements := v.Elements()
		result := make([]any, len(elements))
		for i, elem := range elements {
			result[i] = FromValue(elem)
		}
		return result
	case types.Map:
		elements := v.Elements()
		result := make(map[string]any)
		for k, elem := range elements {
			result[k] = FromValue(elem)
		}
		return result
	case types.Object:
		attrs := v.Attributes()
		result := make(map[string]any)
		for k, attr := range attrs {
			result[k] = FromValue(attr)
		}
		return result
	default:
		return fmt.Sprintf("%+v", v)
	}
}

// FromValue converts a Terraform attr.Value to a native Go type.
// It handles null values, Dynamic types, primitives, and complex types recursively.
//
// Parameters:
//   - in: The Terraform attr.Value to convert
//
// Returns a Go value of the appropriate type:
//   - nil for null values
//   - Recursively converts Dynamic values via FromDynamic
//   - string for String values
//   - bool for Bool values
//   - float64 for Number values
//   - []any for List values (with recursive element conversion)
//   - map[string]any for Map and Object values (with recursive element conversion)
//   - string representation for unknown types
func FromValue(in attr.Value) any {
	if in.IsNull() {
		return nil
	}

	switch v := in.(type) {
	case types.Dynamic:
		return FromDynamic(v)
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		bigFloat := v.ValueBigFloat()
		if bigFloat != nil {
			f64, _ := bigFloat.Float64()
			return f64
		}
		return nil
	case types.List:
		elements := v.Elements()
		result := make([]any, len(elements))
		for i, elem := range elements {
			result[i] = FromValue(elem)
		}
		return result
	case types.Map:
		elements := v.Elements()
		result := make(map[string]any)
		for k, elem := range elements {
			result[k] = FromValue(elem)
		}
		return result
	case types.Object:
		attrs := v.Attributes()
		result := make(map[string]any)
		for k, attr := range attrs {
			result[k] = FromValue(attr)
		}
		return result
	default:
		return fmt.Sprintf("%+v", v)
	}
}

// ToDynamic converts a native Go value to a Terraform Dynamic type.
// It handles nil values, pointer dereferencing, primitives, and complex types.
//
// Parameters:
//   - value: The Go value to convert (supports any, but specific types are handled specially)
//
// Returns a types.Dynamic value:
//   - types.DynamicNull() for nil values
//   - Automatically dereferences pointers before conversion
//   - Converts string, bool, numeric types to appropriate Terraform types
//   - Converts []any to types.List with Dynamic elements
//   - Converts map[string]any to types.Object with Dynamic values
//   - Falls back to string representation for unknown types
//
// Supported numeric types: float64, float32, int, int64, int32.
func ToDynamic(value any) types.Dynamic {
	if value == nil {
		return types.DynamicNull()
	}

	// Dereference pointers
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return types.DynamicNull()
		}
		rv = rv.Elem()
	}
	value = rv.Interface()

	switch v := value.(type) {
	case string:
		return types.DynamicValue(types.StringValue(v))
	case bool:
		return types.DynamicValue(types.BoolValue(v))
	case float64:
		numVal := types.NumberValue(big.NewFloat(v))
		return types.DynamicValue(numVal)
	case float32:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int64:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int32:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case []any:
		elements := make([]attr.Value, len(v))
		for i, elem := range v {
			elements[i] = ToDynamic(elem)
		}
		listVal, _ := types.ListValue(types.DynamicType, elements)
		return types.DynamicValue(listVal)
	case map[string]any:
		// Convert map to object instead of map to support mixed types
		elements := make(map[string]attr.Value)
		attrTypes := make(map[string]attr.Type)
		for k, elem := range v {
			dynValue := ToDynamic(elem)
			elements[k] = dynValue
			attrTypes[k] = types.DynamicType
		}
		objVal, _ := types.ObjectValue(attrTypes, elements)
		return types.DynamicValue(objVal)
	default:
		// Fallback: convert to string
		return types.DynamicValue(types.StringValue(fmt.Sprintf("%+v", v)))
	}
}
