package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func int64Value(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok && v != nil {
		switch t := v.(type) {
		case float64:
			return int64(t)
		case int64:
			return t
		case int:
			return int64(t)
		}
	}
	return 0
}

func boolValue(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok && v != nil {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			return t == "True" || t == "true"
		}
	}
	return false
}

func listStrings(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var items []string
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	return items
}

func stringSliceEqual(a, b []string) bool {
	return reflect.DeepEqual(a, b)
}
