function(env) [
    {[if std.objectHas(env, "include") then 'include']: env.include},
    {[if std.objectHas(env, "passwd") then 'passwd']: env.passwd},
    {[if std.objectHas(env, "files") then 'files']: env.files},
    {[if std.objectHas(env, "systemd") then 'systemd']: env.systemd},
    {[if std.objectHas(env, "networkd") then 'networkd']: env.networkd}
]
