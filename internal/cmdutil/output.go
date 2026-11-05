package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

// JSONFlag accepts --json with an optional comma-separated field list.
type JSONFlag struct {
	Enabled bool
	Fields  []string
}

// String implements pflag.Value.
func (j *JSONFlag) String() string {
	if !j.Enabled {
		return ""
	}
	if len(j.Fields) == 0 {
		return "all"
	}
	return strings.Join(j.Fields, ",")
}

// Set implements pflag.Value.
func (j *JSONFlag) Set(value string) error {
	j.Enabled = true
	j.Fields = nil
	if value == "" || value == "all" {
		return nil
	}
	for _, field := range strings.Split(value, ",") {
		field = strings.TrimSpace(field)
		if field != "" {
			j.Fields = append(j.Fields, field)
		}
	}
	return nil
}

// Type implements pflag.Value.
func (j *JSONFlag) Type() string { return "fields" }

// OutputFlags controls structured output for a command.
type OutputFlags struct {
	JSON     JSONFlag
	Template string
}

// AddOutputFlags registers --json and --template on cmd.
func AddOutputFlags(cmd *cobra.Command) *OutputFlags {
	o := &OutputFlags{}
	cmd.Flags().Var(&o.JSON, "json", "Output JSON; use --json=FIELD,... to select fields")
	if flag := cmd.Flags().Lookup("json"); flag != nil {
		flag.NoOptDefVal = "all"
	}
	cmd.Flags().StringVar(&o.Template, "template", "", "Format output with a Go template")
	return o
}

// IsJSON reports whether JSON output was requested.
func (o *OutputFlags) IsJSON() bool { return o.JSON.Enabled }

// Print writes v as filtered JSON or executes the template.
func (o *OutputFlags) Print(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return err
	}
	if len(o.JSON.Fields) > 0 {
		generic = filterFields(generic, o.JSON.Fields)
	}
	if o.Template != "" {
		tmpl, err := template.New("output").Funcs(template.FuncMap{
			"join": strings.Join,
			"json": func(value any) string {
				b, _ := json.Marshal(value)
				return string(b)
			},
		}).Parse(o.Template)
		if err != nil {
			return fmt.Errorf("invalid template: %w", err)
		}
		return tmpl.Execute(w, generic)
	}
	b, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func filterFields(value any, fields []string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(fields))
		for _, field := range fields {
			if v, ok := typed[field]; ok {
				out[field] = v
			}
		}
		return out
	case []any:
		for i := range typed {
			typed[i] = filterFields(typed[i], fields)
		}
		return typed
	default:
		return value
	}
}
