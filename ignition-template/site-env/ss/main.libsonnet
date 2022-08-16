local config_template = import '../../config.libsonnet';
local utility = import '../../utility.libsonnet';

function(settings)
    utility.union_map(std.map(function(x) { [if x=="base" then 'site.yml' else 'site-'+x +'.yml']: config_template(settings.ss[x]) }, utility.get_ss(settings)))
