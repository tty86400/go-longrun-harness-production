package provider

import "testing"

func TestExtractJSONObjectText(t *testing.T) {
    cases := []struct {
        name string
        in   string
    }{
        {name: "plain", in: `{"a":1,"b":"x"}`},
        {name: "fenced", in: "```json\n{\"a\":1}\n```"},
        {name: "wrapped", in: "Here is the result:\n{\"a\":1,\"b\":{\"c\":2}}\nThanks"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            out, err := ExtractJSONObjectText(tc.in)
            if err != nil {
                t.Fatalf("unexpected err: %v", err)
            }
            if out == "" {
                t.Fatalf("expected non-empty JSON")
            }
        })
    }
}
