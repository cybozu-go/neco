{
  // union_map transforms
  // [
  //   { "a": "value a" },
  //   { "b": "value b" },
  // ]
  // into
  // {
  //   "a": "value a",
  //   "b": "value b",
  // }
  union_map(arr)::
    std.foldl(function(x, y) x + y, arr, {}),

  // prefix_file_names_array transforms
  // {
  //   "path/to/file1.yaml": "file 1 content in JSON",
  //   "path/to/file2.yaml": "file 2 content in JSON"
  // }
  // into
  // [
  //   { "prefix/path/to/file1.yaml": "file 1 content in JSON" },
  //   { "prefix/path/to/file2.yaml": "file 2 content in JSON" },
  // ]
  prefix_file_names_array(prefix, files)::
    std.objectValues(std.mapWithKey(function(x, y) { [prefix + '/' + x]: y }, files)),

  // prefix_file_names transforms
  // {
  //   "path/to/file1.yaml": "file 1 content in JSON",
  //   "path/to/file2.yaml": "file 2 content in JSON"
  // }
  // into
  // {
  //   "prefix/path/to/file1.yaml": "file 1 content in JSON",
  //   "prefix/path/to/file2.yaml": "file 2 content in JSON"
  // }
  prefix_file_names(prefix, files)::
    self.union_map(self.prefix_file_names_array(prefix, files)),

  // get_migrating_teams retrieves the array of migrating team from settings.
  get_migrating_teams(settings)::
    std.filter(function(x) std.count(settings.migrating_teams, x) > 0, self.get_teams(settings)),

  // get_tenants retrieves the array of tenants from settings.
  get_tenants(settings)::
    std.objectFields(settings.tenants),
  
  // get_boots retrives the array of boot from settings.
  get_boot(settings)::
    std.objectFields(settings.boot),
  
  // get_cs retrives the array of cs from settings.
  get_cs(settings)::
    std.objectFields(settings.cs),
  
  // get_ss retrives the array of ss from settings.
  get_ss(settings)::
    std.objectFields(settings.ss),

  // get_common retrives the array of common from settings.
  get_common(settings)::
    std.objectFields(settings.common),

  // get_destination_apps retrieves the array of tenant apps for the specified destination.
  get_destination_apps(settings, destination)::
    std.filter(function(x) std.objectHas(self.get_app(settings, x).destinations, destination), self.get_apps(settings)),

  // get_app retrieves a tenant app settings.
  get_app(settings, name)::
    settings.apps[name],
}
