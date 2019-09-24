# README

```bash
go-nhc \
  --interface ib0 \
  --interface eno1 \
  --interface lo \
  --infiniband mlx5_0:1=100 \
  --mount / \
  --mount /boot \
  --mount /nec/vol1=nec_vol1=gpfs \
  --file /nec/vol1/testfile \
  --mount /vsc-hard-mounts/leuven-apps=10.118.240.67:/apps/tier1/2016=nfs4 \
  --mount /vsc-hard-mounts/leuven-user=10.118.240.67:/user=nfs4 \
  --mount /vsc-hard-mounts/leuven-data=10.118.240.67:/data=nfs4 \
  --mount /local_scratch=/dev/sda5=ext4 \
  --memory 1024B \
  --total-memory 1024B \
  --dimms consistent \
  --hyperthreading disabled \
  --cpu-sockets 2 \
  --disk-usage /local_scratch=98% \
  --disk-usage /=85% \
  --disk-usage /boot=40MB \
  --disk-usage /home=98% \
  --disk-usage /tmp=98% \
  --user vsc30001
```
