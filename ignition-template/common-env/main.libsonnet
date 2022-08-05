local utility = import '../utility.libsonnet';
local config_template = import '../config.libsonnet';

function(settings)
    utility.union_map(std.map(function(x) { [if x=="common" then 'common.yml' else 'common-'+x +'.yml']: config_template(settings.common[x]) }, utility.get_common(settings)))
