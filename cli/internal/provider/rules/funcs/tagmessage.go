package funcs

import "sync"

// tagMessageRegistry holds error messages for custom validate tags registered
// through ConfigValidateFuncs. Those tags carry no parameters for the message
// formatter to work from, so without an entry here they fall through to the
// raw go-playground error, which exposes internal struct names.
type tagMessageRegistry struct {
	messages map[string]string
	mu       sync.RWMutex
}

var tagMessages = &tagMessageRegistry{
	messages: make(map[string]string),
}

// NewTagMessage registers the error message for a custom validate tag. The
// message is appended to the field name, so it reads as a predicate:
// NewTagMessage("my_tag", "is required when 'other' is set").
func NewTagMessage(tag, message string) {
	tagMessages.mu.Lock()
	defer tagMessages.mu.Unlock()

	if tagMessages.messages == nil {
		tagMessages.messages = make(map[string]string)
	}
	tagMessages.messages[tag] = message
}

// getTagErrorMessage retrieves the message registered for a custom tag.
func getTagErrorMessage(tag string) (string, bool) {
	tagMessages.mu.RLock()
	defer tagMessages.mu.RUnlock()

	msg, ok := tagMessages.messages[tag]
	return msg, ok
}
