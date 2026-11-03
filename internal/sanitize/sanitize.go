// Package sanitize normalises user input before it is validated or stored. Request structs
// declare the rules per field with a `sanitize` tag, applied in order by Struct:
//
//	Email    string  `json:"email" sanitize:"trim,lower"`
//	Name     *string `json:"name" sanitize:"trim,collapse,capfirst"`
//	Password string  `json:"password" sanitize:"-"` // deliberately kept as typed
//
// Values that are not request-struct fields (form values, query parameters) call the rule
// functions directly. See docs/conventions.md → Input sanitisation.
package sanitize

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TagName is the struct tag Struct reads.
const TagName = "sanitize"

// rules maps a tag rule name to its function.
var rules = map[string]func(string) string{
	"trim":     Trim,
	"lower":    Lower,
	"collapse": Collapse,
	"capfirst": CapFirst,
}

// isBlank reports whether r is whitespace or an invisible character that pasted text
// commonly carries along (zero-width space, byte order mark).
func isBlank(r rune) bool {
	return unicode.IsSpace(r) || r == '\u200b' || r == '\ufeff'
}

// Trim removes leading and trailing whitespace, including Unicode spaces such as the
// non-breaking space, and zero-width characters.
func Trim(value string) string {
	return strings.TrimFunc(value, isBlank)
}

// Lower lowercases the whole value.
func Lower(value string) string {
	return strings.ToLower(value)
}

// Collapse trims the value and squashes every inner run of whitespace into one space.
// It also joins lines, so it is only for single-line values.
func Collapse(value string) string {
	return strings.Join(strings.FieldsFunc(value, isBlank), " ")
}

// CapFirst uppercases the first letter and leaves the rest untouched ("easy run" →
// "Easy run", "iPhone" → "IPhone"). Meant for short labels, never for people's names.
func CapFirst(value string) string {
	first, size := utf8.DecodeRuneInString(value)
	if first == utf8.RuneError || unicode.IsUpper(first) {
		return value
	}
	return string(unicode.ToUpper(first)) + value[size:]
}

// Email trims and lowercases an e-mail address.
func Email(value string) string {
	return Lower(Trim(value))
}

// Struct applies the `sanitize` tag rules to every tagged field of the struct target
// points to, recursing into nested structs, pointers and slices. Tagged fields may be a
// string, *string, []string or *[]string; nil pointers stay nil. A target that is not a
// pointer to a struct (e.g. a bound []string) is left alone.
//
// An unknown rule, or a rule on a field that holds no strings, panics: both are
// programming errors that the first test touching the struct should surface.
func Struct(target any) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return
	}
	walk(value.Elem())
}

func walk(value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			walk(value.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			walk(value.Index(i))
		}
	case reflect.Struct:
		structType := value.Type()
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if !field.IsExported() {
				continue
			}
			tag, tagged := field.Tag.Lookup(TagName)
			if !tagged {
				walk(value.Field(i))
				continue
			}
			if tag == "-" {
				continue
			}
			apply(value.Field(i), parseRules(structType, field.Name, tag))
		}
	}
}

// parseRules resolves a tag into its rule functions, panicking on unknown names.
func parseRules(structType reflect.Type, fieldName, tag string) []func(string) string {
	var resolved []func(string) string
	for _, name := range strings.Split(tag, ",") {
		rule, ok := rules[strings.TrimSpace(name)]
		if !ok {
			panic(fmt.Sprintf("sanitize: unknown rule %q on %s.%s", name, structType, fieldName))
		}
		resolved = append(resolved, rule)
	}
	return resolved
}

// apply runs the rules over a string-holding field.
func apply(field reflect.Value, fieldRules []func(string) string) {
	switch field.Kind() {
	case reflect.String:
		if !field.CanSet() {
			return
		}
		result := field.String()
		for _, rule := range fieldRules {
			result = rule(result)
		}
		field.SetString(result)
	case reflect.Pointer:
		if !field.IsNil() {
			apply(field.Elem(), fieldRules)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < field.Len(); i++ {
			apply(field.Index(i), fieldRules)
		}
	default:
		panic(fmt.Sprintf("sanitize: tag on a %s field, only strings can be sanitised", field.Type()))
	}
}
