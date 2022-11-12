package module

import (
	"bytes"
	"text/template"
)

func init() {
	register("template", func(ctx Context) interface{} {
		return func(name string, input map[string]interface{}) (string, error) {
			var content string
			if err := ctx.Db.QueryRow("select content from source where name = ? and type = 'template' and active = true", name).Scan(&content); err != nil {
				return "", err
			}

			t, err := template.New(name).Parse(content)
			if err != nil {
				return "", err
			}
			buf := new(bytes.Buffer)
			if err := t.Execute(buf, input); err != nil {
				return "", err
			}
			return buf.String(), nil
		}
	})
}
