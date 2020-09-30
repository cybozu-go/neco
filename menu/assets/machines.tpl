racks:
{{range $rack := .Racks -}}
- name: {{$rack.Name}}
  workers:
    cp: {{len $rack.CPList }}
    cs: {{len $rack.CSList }}
    ss: {{len $rack.SSList }}
  boot:
    bastion: {{$rack.BootNode.BastionAddress}}
{{end -}}
