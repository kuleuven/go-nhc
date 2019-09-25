# README

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
  --user vsc30001
```
