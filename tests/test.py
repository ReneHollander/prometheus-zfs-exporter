import re


def get_value(res, s):
    o = re.findall(rf"^{re.escape(s)} (.*)$", res, re.MULTILINE)
    if len(o) == 0:
        raise AssertionError(f'metric "{s}" not found in input:\n{res}')
    return float(o[0])


start_all()

# Set up ZFS pool and a dataset
machine.succeed(
    "modprobe zfs",
    "udevadm settle",
    "parted --script /dev/vdb mklabel msdos",
    "parted --script /dev/vdb -- mkpart primary 1024M -1s",
    "udevadm settle",
    "zpool create dpool /dev/vdb1",
    "zfs create -o mountpoint=/mnt dpool/data",
    "echo 'test' > /mnt/test",
    "cat /mnt/test",
)

machine.wait_for_unit("prometheus-zfs-exporter.service")
machine.wait_for_open_port(9901)

res = machine.succeed("curl http://127.0.0.1:9901/metrics")
print(res)

# Check some basic pool metrics
assert get_value(res, 'zfs_pool_state{pool="dpool",state="ACTIVE"}') == 1
assert (
    get_value(
        res,
        'zfs_pool_vdev_state{pool="dpool",state="ONLINE",vdev="dpool",vdev_type="root"}',
    )
    == 1
)
assert (
    get_value(
        res, 'zfs_pool_vdev_write_ops{pool="dpool",vdev="dpool",vdev_type="root"}'
    )
    > 0
)
assert (
    get_value(
        res, 'zfs_pool_vdev_write_ops{pool="dpool",vdev="dpool/vdb1",vdev_type="disk"}'
    )
    > 0
)

# Check some basic dataset metrics
assert get_value(res, 'zfs_dataset_available{name="dpool/data",pool="dpool"}') > 0
assert get_value(res, 'zfs_dataset_writes{name="dpool/data",pool="dpool"}') > 0
assert get_value(res, 'zfs_dataset_nwritten{name="dpool/data",pool="dpool"}') > 0
assert get_value(res, 'zfs_dataset_reads{name="dpool/data",pool="dpool"}') > 0
assert get_value(res, 'zfs_dataset_nread{name="dpool/data",pool="dpool"}') > 0
