// Package api implements the raw REST escape hatch.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
)

// NewCmdAPI returns the api command.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	var methodFlag, input string
	var fields, rawFields, headerFlags []string
	var paginate, silent, include bool

	cmd := &cobra.Command{
		Use:   "api [METHOD] PATH",
		Short: "Make a raw Bitbucket REST API request",
		Long: `Send an authenticated request to the Bitbucket REST API.

The path is relative to the configured API base (/2.0 on Cloud,
/rest/api/1.0 on Data Center). Use this command for endpoints bk does not wrap.

Examples:
  bk api /user
  bk api /repositories/myteam/myrepo/pullrequests -f state=OPEN
  bk api POST /repositories/myteam/myrepo/commit/abc123/statuses/build \
    -f key=CI -f state=SUCCESSFUL -f url=https://ci.example.com/1
  bk api /repositories/myteam/myrepo/pullrequests --paginate`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}

			method := strings.ToUpper(methodFlag)
			path := ""
			if len(args) == 2 {
				if method != "" {
					return fmt.Errorf("pass the method either as the first argument or with --method, not both")
				}
				method = strings.ToUpper(args[0])
				path = args[1]
			} else {
				path = args[0]
			}
			if method == "" {
				method = http.MethodGet
			}
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
			default:
				return fmt.Errorf("unsupported method %q", method)
			}
			if path == "" {
				return fmt.Errorf("path is required")
			}

			headers := http.Header{}
			for _, header := range headerFlags {
				key, value, ok := strings.Cut(header, ":")
				if !ok {
					return fmt.Errorf("invalid header %q: expected 'Name: value'", header)
				}
				headers.Add(strings.TrimSpace(key), strings.TrimSpace(value))
			}

			fieldMap, err := cmdutil.ParseStringFields(fields)
			if err != nil {
				return err
			}
			rawMap, err := cmdutil.ParseRawFields(rawFields)
			if err != nil {
				return err
			}
			mergeMaps(fieldMap, rawMap)

			query := url.Values{}
			var body []byte
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodDelete:
				if input != "" {
					return fmt.Errorf("--input is only valid for POST, PUT, and PATCH")
				}
				for key, value := range fieldMap {
					query.Set(key, fmt.Sprint(value))
				}
			default:
				if input != "" {
					if len(fieldMap) > 0 {
						return fmt.Errorf("--input cannot be combined with --field or --raw-field")
					}
					data, err := readInput(input, app.IO.In)
					if err != nil {
						return err
					}
					body = data
				} else if len(fieldMap) > 0 {
					data, err := json.Marshal(fieldMap)
					if err != nil {
						return err
					}
					body = data
				}
			}
			path = appendQuery(path, query)

			if paginate {
				if method != http.MethodGet {
					return fmt.Errorf("--paginate requires GET")
				}
				items, err := bitbucket.GetAll[json.RawMessage](ctx, app.Client, path, nil, 0)
				if err != nil {
					return err
				}
				if silent {
					return nil
				}
				return writeJSON(app.IO.Out, items)
			}

			resp, data, err := app.Client.Do(ctx, method, path, body, "application/json", headers)
			if err != nil {
				return err
			}
			if include && resp != nil {
				fmt.Fprintf(app.IO.Out, "HTTP %s\n", resp.Status)
				keys := make([]string, 0, len(resp.Header))
				for key := range resp.Header {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					for _, value := range resp.Header.Values(key) {
						fmt.Fprintf(app.IO.Out, "%s: %s\n", key, value)
					}
				}
				fmt.Fprintln(app.IO.Out)
			}
			if silent {
				return nil
			}
			if len(bytes.TrimSpace(data)) == 0 {
				return nil
			}
			if json.Valid(data) {
				var out bytes.Buffer
				if err := json.Indent(&out, data, "", "  "); err != nil {
					return err
				}
				_, err := fmt.Fprintln(app.IO.Out, out.String())
				return err
			}
			if _, err := app.IO.Out.Write(data); err != nil {
				return err
			}
			if !bytes.HasSuffix(data, []byte("\n")) {
				fmt.Fprintln(app.IO.Out)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&methodFlag, "method", "X", "", "HTTP method")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Read the request body from a file (- for stdin)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Add a string field (KEY=VALUE, repeatable)")
	cmd.Flags().StringArrayVarP(&rawFields, "raw-field", "F", nil, "Add a typed JSON field (KEY=JSON, repeatable)")
	cmd.Flags().StringArrayVarP(&headerFlags, "header", "H", nil, "Add a request header (repeatable)")
	cmd.Flags().BoolVar(&paginate, "paginate", false, "Follow pagination and print all results as a JSON array")
	cmd.Flags().BoolVar(&silent, "silent", false, "Do not print the response body")
	cmd.Flags().BoolVar(&include, "include", false, "Print the response status and headers")
	return cmd
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(io.LimitReader(stdin, 16<<20))
	}
	return os.ReadFile(path)
}

func appendQuery(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + values.Encode()
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func mergeMaps(dst, src map[string]any) {
	for key, value := range src {
		existing, ok := dst[key].(map[string]any)
		incoming, isMap := value.(map[string]any)
		if ok && isMap {
			mergeMaps(existing, incoming)
			continue
		}
		dst[key] = value
	}
}
