package b

import (
	"os"
	"text/template"
)

func main() {
	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
	</body>
</html>`

	t, _ := template.New("webpage").Parse(tpl)

	data := struct {
		Title string
	}{
		Title: "My page",
	}

	t.Execute(os.Stdout, data)
}
