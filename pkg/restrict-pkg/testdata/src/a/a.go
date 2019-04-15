package a

import (
	"html/template" // want "html/template package must not be imported"
	"os"
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
