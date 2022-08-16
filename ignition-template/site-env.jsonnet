local site_env_template = import 'site-env/main.libsonnet';
local settings = import 'settings.json';
local utility = import 'utility.libsonnet';

utility.prefix_file_names('ignitions/roles', site_env_template(settings)) 
