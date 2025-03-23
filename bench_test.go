package main

import (
	"io"
	"testing"

	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/ioctl"
	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/kstat"
	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/nvlist"
)

func BenchmarkCollect(b *testing.B) {
	zfsHandle, err := ioctl.NewZFSHandle()
	if err != nil {
		b.Fatal(err)
	}

	c := &zfsCollector{zfsHandle: zfsHandle}
	c.describe(nil)

	for b.Loop() {
		c.collect(nil)
	}
}

func BenchmarkDecode(b *testing.B) {
	zfsHandle, err := ioctl.NewZFSHandle()
	if err != nil {
		b.Fatal(err)
	}

	var handle func(r *nvlist.NVListReader) error
	handle = func(r *nvlist.NVListReader) error {
		for {
			token, err := r.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			if token == nvlist.TypeNvlist {
				err = handle(r)
				if err != nil {
					return err
				}
			} else if token == nvlist.TypeNvlistArray {
				numElements := r.NumElements()
				for range numElements {
					err = handle(r)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	cmd := ioctl.Cmd{}
	resp := make([]byte, 256*1024)
	err = zfsHandle.Ioctl(ioctl.ZFS_IOC_POOL_CONFIGS, &cmd, nil, nil, &resp)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		r := nvlist.NVListReader{
			Data: resp,
		}
		err = handle(&r)
		if err != nil {
			b.Fatal(err)
		}
	}

}

func BenchmarkDecodeKStat(b *testing.B) {
	kstatData := []byte(`182 1 0x01 27 7600 8327482934 352277655959
name                            type data
dataset_name                    7    rpool/safe/home
writes                          4    52239
nwritten                        4    609481617
reads                           4    252308
nread                           4    5456852343
nunlinks                        4    731
nunlinked                       4    729
zil_commit_count                4    2824
zil_commit_writer_count         4    2819
zil_commit_error_count          4    0
zil_commit_stall_count          4    0
zil_commit_suspend_count        4    0
zil_itx_count                   4    16522
zil_itx_indirect_count          4    3073
zil_itx_indirect_bytes          4    186049521
zil_itx_copied_count            4    0
zil_itx_copied_bytes            4    0
zil_itx_needcopy_count          4    10492
zil_itx_needcopy_bytes          4    32314808
zil_itx_metaslab_normal_count   4    2414
zil_itx_metaslab_normal_bytes   4    35507144
zil_itx_metaslab_normal_write   4    43057152
zil_itx_metaslab_normal_alloc   4    61841408
zil_itx_metaslab_slog_count     4    0
zil_itx_metaslab_slog_bytes     4    0
zil_itx_metaslab_slog_write     4    0
zil_itx_metaslab_slog_alloc     4    0
`)

	for b.Loop() {
		r := kstat.KStatReader{
			Data: kstatData,
		}
		for {
			_, err := r.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				b.Fatal(err)
			}
		}
	}

}
