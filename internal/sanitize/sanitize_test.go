package sanitize

import "testing"

func TestRules(t *testing.T) {
	cases := []struct {
		name  string
		rule  func(string) string
		input string
		want  string
	}{
		{"trim spaces", Trim, "  hello  ", "hello"},
		{"trim tabs and newlines", Trim, "\t\nhello\r\n", "hello"},
		{"trim non-breaking space", Trim, "\u00a0hello\u00a0", "hello"},
		{"trim ideographic space", Trim, "\u3000hello\u2003", "hello"},
		{"trim zero-width and BOM", Trim, "\ufeff\u200bhello\u200b", "hello"},
		{"trim keeps inner spaces", Trim, " a  b ", "a  b"},
		{"trim empty", Trim, "", ""},
		{"trim only blanks", Trim, " \t\u00a0 ", ""},
		{"lower", Lower, "Ola@Example.COM", "ola@example.com"},
		{"lower non-ASCII", Lower, "ÆØÅ", "æøå"},
		{"collapse", Collapse, "  easy \t  run\n ", "easy run"},
		{"collapse non-breaking", Collapse, "a\u00a0\u00a0b", "a b"},
		{"collapse empty", Collapse, "   ", ""},
		{"capfirst", CapFirst, "easy run", "Easy run"},
		{"capfirst leaves the rest", CapFirst, "iPhone pro", "IPhone pro"},
		{"capfirst already capital", CapFirst, "Easy RUN", "Easy RUN"},
		{"capfirst æ", CapFirst, "ærlig", "Ærlig"},
		{"capfirst ø", CapFirst, "østmarka", "Østmarka"},
		{"capfirst å", CapFirst, "åsen", "Åsen"},
		{"capfirst digit", CapFirst, "5k shoes", "5k shoes"},
		{"capfirst empty", CapFirst, "", ""},
		{"email", Email, "  Ola@Example.COM \t", "ola@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule(tc.input); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

type inner struct {
	Label string `sanitize:"trim,capfirst"`
	Raw   string
}

type sample struct {
	Email     string  `sanitize:"trim,lower"`
	Name      *string `sanitize:"trim,collapse,capfirst"`
	Nil       *string `sanitize:"trim"`
	Password  string  `sanitize:"-"`
	Untagged  string
	IDs       []string  `sanitize:"trim"`
	Tags      *[]string `sanitize:"trim,lower"`
	Inner     inner
	InnerPtr  *inner
	Inners    []inner
	NoStrings int
	private   string `sanitize:"trim"`
}

func TestStruct(t *testing.T) {
	name := "  easy   run "
	tags := []string{" Long ", "EASY"}
	value := sample{
		Email:    "  Ola@Example.COM ",
		Name:     &name,
		Password: "  secret  ",
		Untagged: "  as typed ",
		IDs:      []string{" a ", "b "},
		Tags:     &tags,
		Inner:    inner{Label: " inner ", Raw: " raw "},
		InnerPtr: &inner{Label: " pointer "},
		Inners:   []inner{{Label: " one "}, {Label: " two "}},
		private:  " private ",
	}

	Struct(&value)

	checks := map[string][2]string{
		"Email":    {value.Email, "ola@example.com"},
		"Name":     {*value.Name, "Easy run"},
		"Password": {value.Password, "  secret  "},
		"Untagged": {value.Untagged, "  as typed "},
		"IDs[0]":   {value.IDs[0], "a"},
		"IDs[1]":   {value.IDs[1], "b"},
		"Tags[0]":  {(*value.Tags)[0], "long"},
		"Tags[1]":  {(*value.Tags)[1], "easy"},
		"Inner":    {value.Inner.Label, "Inner"},
		"InnerRaw": {value.Inner.Raw, " raw "},
		"InnerPtr": {value.InnerPtr.Label, "Pointer"},
		"Inners0":  {value.Inners[0].Label, "One"},
		"Inners1":  {value.Inners[1].Label, "Two"},
		"private":  {value.private, " private "},
	}
	for field, pair := range checks {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", field, pair[0], pair[1])
		}
	}
	if value.Nil != nil {
		t.Errorf("nil pointer was set to %q", *value.Nil)
	}
}

func TestStructIgnoresNonStructTargets(t *testing.T) {
	ids := []string{" a "}
	Struct(&ids)
	Struct(nil)
	Struct(sample{})
	var nilPointer *sample
	Struct(nilPointer)
	if ids[0] != " a " {
		t.Errorf("bare slice was changed to %q", ids[0])
	}
}

func TestStructPanicsOnUnknownRule(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic for an unknown rule")
		}
	}()
	value := struct {
		Name string `sanitize:"trim,titel"`
	}{Name: "x"}
	Struct(&value)
}

func TestStructPanicsOnNonStringField(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic for a tag on an int field")
		}
	}()
	value := struct {
		Count int `sanitize:"trim"`
	}{Count: 1}
	Struct(&value)
}
