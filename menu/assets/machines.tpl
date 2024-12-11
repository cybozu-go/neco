racks:
{{range $rack := .Racks -}}
- name: {{$rack.Name}}
  workers:
    cs: {{len $rack.CSList }}
    ss: {{len $rack.SSList }}
    ss2: {{len $rack.SS2List }}
  boot:
    bastion: {{$rack.BootNode.BastionAddress}}
{{end -}}
