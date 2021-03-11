# README

Golang script mimicking the features of nhc, but without spawning subprocesses.

Example usage of the go binary:

```bash
go-nhc \
  --interface ib0 \
  --interface eno1 \
  --interface lo \
  --infiniband 'device=mlx5_0 port=1 speed=100' \
  --mount / \
  --mount /boot \
  --mount '/nec/vol1 device=nec_vol1 fs_type=gpfs' \
  --file /nec/vol1/testfile \
  --mount '/vsc-hard-mounts/leuven-apps device=10.118.240.67:/apps/tier1/2016 fs_type=nfs4' \
  --mount '/vsc-hard-mounts/leuven-user device=10.118.240.67:/user fs_type=nfs4' \
  --mount '/vsc-hard-mounts/leuven-data device=10.118.240.67:/data fs_type=nfs4' \
  --mount '/local_scratch device=/dev/sda5 fs_type=ext4' \
  --memory 1024B \
  --total-memory 1024B \
  --dimms consistent \
  --hyperthreading disabled \
  --cpu-sockets 2 \
  --disk-usage '/local_scratch max_used_percent=98' \
  --disk-usage '/ max_used_percent=85' \
  --disk-usage '/boot min_free=40MB' \
  --disk-usage '/home max_used_percent=98' \
  --disk-usage '/tmp max_used_percent=98' \
  --user vsc30001 \
  --process 'pbs_mom user=root start=yes' \
  --process 'trqauthd user=root start=yes' \
  --process 'sshd user=root start=yes' \
  --process 'polkitd user=polkitd start=yes' \
  --process 'ntpd start=yes' \
  --process 'sssd user=root start=yes' \
  --process 'nscd user=nscd start=yes' \
  --process 'crond user=root start=yes' \
  --process 'nfslock daemon=rpc.statd restart=yes' \
  --process 'rpcbind restart=yes' \
  --process 'autofs daemon=automount user=root start=yes' \
  --process 'postfix daemon=qmgr restart=yes' \
  --unauthorized 'pbs max_system_uid=9000'
```

Check `go-nhc --help` for all options. The above options define all checks, the following additional options can be used to tweak the behaviour of `go-nhc`:

* `-v`: verbose mode - show ignored checks and print summarizing message
* `-l`: list all checks that passed
* `-a`: run all checks, do not stop on first fatal check
* `-s`: do not try to send results to sensu agent or statsd server

## Wrapper using /etc/nhc.conf

The script in `src/usr/bin/nhc` can be used as a wrapper around go-nhc, to read the check definitions from `/etc/nhc.conf`. The above call would correspond to the following conf file:

```
interface ib0
interface eno1
interface lo
infiniband 'device=mlx5_0 port=1 speed=100'
mount /
mount /boot
mount '/nec/vol1 device=nec_vol1 fs_type=gpfs'
file /nec/vol1/testfile
mount '/vsc-hard-mounts/leuven-apps device=10.118.240.67:/apps/tier1/2016 fs_type=nfs4'
mount '/vsc-hard-mounts/leuven-user device=10.118.240.67:/user fs_type=nfs4'
mount '/vsc-hard-mounts/leuven-data device=10.118.240.67:/data fs_type=nfs4'
mount '/local_scratch device=/dev/sda5 fs_type=ext4'
memory 1024B
total-memory 1024B
dimms consistent
hyperthreading disabled
cpu-sockets 2
disk-usage '/local_scratch max_used_percent=98'
disk-usage '/ max_used_percent=85'
disk-usage '/boot min_free=40MB'
disk-usage '/home max_used_percent=98'
disk-usage '/tmp max_used_percent=98'
user vsc30001
process 'pbs_mom user=root start=yes'
process 'trqauthd user=root start=yes'
process 'sshd user=root start=yes'
process 'polkitd user=polkitd start=yes'
process 'ntpd start=yes'
process 'sssd user=root start=yes'
process 'nscd user=nscd start=yes'
process 'crond user=root start=yes'
process 'nfslock daemon=rpc.statd restart=yes'
process 'rpcbind restart=yes'
process 'autofs daemon=automount user=root start=yes'
process 'postfix daemon=qmgr restart=yes'
unauthorized 'pbs max_system_uid=9000'
```

The script `/usr/bin/nhc` will then accept the followin useful options:

* `-v`: verbose mode - show ignored checks and print summarizing message
* `-l`: list all checks that passed
* `-a`: run all checks, do not stop on first fatal check
* `-s`: do not try to send results to sensu agent or statsd server

## Sensu/Statsd integration

By default, `go-nhc` will send results of each check as passive check result to the local sensu-agent, and statistics will be sent to the sensu-agent statsd port. To disable this behaviour, add the `-s` option or edit the code :-).
