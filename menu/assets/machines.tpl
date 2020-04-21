racks:
{{range $rack := .Racks -}}
- name: {{$rack.Name}}
  workers:
    cs: {{len $rack.CSList }}
    ss: {{len $rack.SSList }}
  boot:
    bastion: {{$rack.BootNode.BastionAddress}}
{{end -}}
