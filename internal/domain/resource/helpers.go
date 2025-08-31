package resource

// GetTagsAsMap returns tags as a map[string]string
// Handles both map[string]string and []string types
func (r *Resource) GetTagsAsMap() map[string]string {
	if r.Tags == nil {
		return make(map[string]string)
	}

	switch tags := r.Tags.(type) {
	case map[string]string:
		return tags
	case []string:
		// Convert []string to map[string]string
		result := make(map[string]string)
		for _, tag := range tags {
			result[tag] = ""
		}
		return result
	case map[string]interface{}:
		// Convert map[string]interface{} to map[string]string
		result := make(map[string]string)
		for k, v := range tags {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
		return result
	default:
		return make(map[string]string)
	}
}

// GetTagsAsSlice returns tags as a []string
// Handles both map[string]string and []string types
func (r *Resource) GetTagsAsSlice() []string {
	if r.Tags == nil {
		return []string{}
	}

	switch tags := r.Tags.(type) {
	case []string:
		return tags
	case map[string]string:
		// Convert map[string]string to []string (keys only)
		result := make([]string, 0, len(tags))
		for k := range tags {
			result = append(result, k)
		}
		return result
	case map[string]interface{}:
		// Convert map[string]interface{} to []string (keys only)
		result := make([]string, 0, len(tags))
		for k := range tags {
			result = append(result, k)
		}
		return result
	default:
		return []string{}
	}
}

// HasTag checks if a resource has a specific tag
func (r *Resource) HasTag(tag string) bool {
	if r.Tags == nil {
		return false
	}

	switch tags := r.Tags.(type) {
	case []string:
		for _, t := range tags {
			if t == tag {
				return true
			}
		}
		return false
	case map[string]string:
		_, exists := tags[tag]
		return exists
	case map[string]interface{}:
		_, exists := tags[tag]
		return exists
	default:
		return false
	}
}
