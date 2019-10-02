sbin_path="/sbin"
if [ -n "${PATH##*${sbin_path}}" -a -n "${PATH##*${sbin_path}:*}" ]; then
    export PATH=$PATH:${sbin_path}
fi

usr_sbin_path="/usr/sbin"
if [ -n "${path##*${usr_sbin_path}}" -a -n "${path##*${usr_sbin_path}:*}" ]; then
    export path=$path:${usr_sbin_path}
fi

usr_local_sbin_path="/usr/local/sbin"
if [ -n "${path##*${usr_local_sbin_path}}" -a -n "${path##*${usr_local_sbin_path}:*}" ]; then
    export path=$path:${usr_local_sbin_path}
fi
