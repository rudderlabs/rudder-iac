package resources

type PropertyRef struct {
	URN      string `json:"urn"`
	Property string `json:"property"`
}

func CollectReferences(v interface{}) []*PropertyRef {
	var refs []*PropertyRef

	switch v := v.(type) {
	case map[string]interface{}:
		for _, vv := range v {
			refs = append(refs, CollectReferences(vv)...)
		}
	case []interface{}:
		for _, vv := range v {
			refs = append(refs, CollectReferences(vv)...)
		}
	case *PropertyRef:
		refs = append(refs, v)
	case PropertyRef:
		refs = append(refs, &v)
	case ResourceData:
		for _, vv := range v {
			refs = append(refs, CollectReferences(vv)...)
		}
	}

	return refs
}
