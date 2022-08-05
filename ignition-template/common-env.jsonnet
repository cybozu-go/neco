local common_env_template = import 'common-env/main.libsonnet';
local settings = import 'settings.json';
local utility = import 'utility.libsonnet';

utility.prefix_file_names('ignitions/common', common_env_template(settings)) 
