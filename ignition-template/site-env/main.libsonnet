local boot_template = import 'boot/main.libsonnet';
local cs_template = import 'cs/main.libsonnet';
local ss_template = import 'ss/main.libsonnet';
local ss2_template = import 'ss2/main.libsonnet';
local utility = import '../utility.libsonnet';
function(settings)
    utility.prefix_file_names('boot', boot_template(settings)) +
    utility.prefix_file_names('cs', cs_template(settings)) +
    utility.prefix_file_names('ss', ss_template(settings)) +
    utility.prefix_file_names('ss2', ss2_template(settings))
