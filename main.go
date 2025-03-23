package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/ioctl"
	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/kstat"
	"github.com/ReneHollander/prometheus-zfs-exporter/zfs/nvlist"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sys/unix"
)

var (
	listenAddr = flag.String("listen-addr", "127.0.0.1:9901", "Address and port to listen on")
)

func describe(ch *chan<- *prometheus.Desc, desc **prometheus.Desc, d *prometheus.Desc) {
	*desc = d
	if ch != nil {
		*ch <- d
	}
}

func export(ch *chan<- prometheus.Metric, desc *prometheus.Desc, valueType prometheus.ValueType, v float64, labels []string) error {
	metric, err := prometheus.NewConstMetric(desc, valueType, v, labels...)
	if err != nil {
		return fmt.Errorf("error exporting metric %v: %w", desc, err)
	}
	if ch != nil {
		*ch <- metric
	}
	return nil
}

type zfsCollector struct {
	zfsHandle *ioctl.ZFSHandle

	poolState      *prometheus.Desc
	poolErrorCount *prometheus.Desc

	poolVdevState          *prometheus.Desc
	poolVdevAllocSpaceDesc *prometheus.Desc
	poolVdevTotalSpaceDesc *prometheus.Desc
	poolVdevDefSpaceDesc   *prometheus.Desc
	poolVdevRepDevSizeDesc *prometheus.Desc
	poolVdevPhysSpace      *prometheus.Desc

	poolVdevReadOps    *prometheus.Desc
	poolVdevReadBytes  *prometheus.Desc
	poolVdevReadErrors *prometheus.Desc

	poolVdevWriteOps    *prometheus.Desc
	poolVdevWriteBytes  *prometheus.Desc
	poolVdevWriteErrors *prometheus.Desc

	poolVdevChechsumErrors *prometheus.Desc
	poolVdevSlowIos        *prometheus.Desc

	datasetAvailable            *prometheus.Desc
	datasetCompressRatio        *prometheus.Desc
	datasetUsed                 *prometheus.Desc
	datasetUsedByChildren       *prometheus.Desc
	datasetUsedByDataset        *prometheus.Desc
	datasetUsedByRefReservation *prometheus.Desc
	datasetUsedBySnapshots      *prometheus.Desc
	datasetReferenced           *prometheus.Desc
	datasetRefCompressRatio     *prometheus.Desc
	datasetLogicalReferenced    *prometheus.Desc
	datasetLogicalUsed          *prometheus.Desc

	datasetWrites    *prometheus.Desc
	datasetNWritten  *prometheus.Desc
	datasetReads     *prometheus.Desc
	datasetNRead     *prometheus.Desc
	datasetUnlinks   *prometheus.Desc
	datasetNUnlinked *prometheus.Desc
}

func (c *zfsCollector) describe(ch *chan<- *prometheus.Desc) {
	describe(ch, &c.poolState, prometheus.NewDesc("zfs_pool_state", "", []string{"pool", "state"}, nil))
	describe(ch, &c.poolErrorCount, prometheus.NewDesc("zfs_pool_error_count", "", []string{"pool"}, nil))

	describe(ch, &c.poolVdevState, prometheus.NewDesc("zfs_pool_vdev_state", "", []string{"pool", "vdev", "vdev_type", "state"}, nil))
	describe(ch, &c.poolVdevAllocSpaceDesc, prometheus.NewDesc("zfs_pool_vdev_alloc_space", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevTotalSpaceDesc, prometheus.NewDesc("zfs_pool_vdev_total_space", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevDefSpaceDesc, prometheus.NewDesc("zfs_pool_vdev_def_space", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevRepDevSizeDesc, prometheus.NewDesc("zfs_pool_vdev_rep_dev_size", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevPhysSpace, prometheus.NewDesc("zfs_pool_vdev_phys_space", "", []string{"pool", "vdev", "vdev_type"}, nil))

	describe(ch, &c.poolVdevReadOps, prometheus.NewDesc("zfs_pool_vdev_read_ops", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevReadBytes, prometheus.NewDesc("zfs_pool_vdev_read_bytes", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevReadErrors, prometheus.NewDesc("zfs_pool_vdev_read_errors", "", []string{"pool", "vdev", "vdev_type"}, nil))

	describe(ch, &c.poolVdevWriteOps, prometheus.NewDesc("zfs_pool_vdev_write_ops", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevWriteBytes, prometheus.NewDesc("zfs_pool_vdev_write_bytes", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevWriteErrors, prometheus.NewDesc("zfs_pool_vdev_write_errors", "", []string{"pool", "vdev", "vdev_type"}, nil))

	describe(ch, &c.poolVdevChechsumErrors, prometheus.NewDesc("zfs_pool_vdev_checksum_errors", "", []string{"pool", "vdev", "vdev_type"}, nil))
	describe(ch, &c.poolVdevSlowIos, prometheus.NewDesc("zfs_pool_vdev_slow_ios", "", []string{"pool", "vdev", "vdev_type"}, nil))

	describe(ch, &c.datasetAvailable, prometheus.NewDesc("zfs_dataset_available", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetCompressRatio, prometheus.NewDesc("zfs_dataset_compress_ratio", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUsed, prometheus.NewDesc("zfs_dataset_used", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUsedByChildren, prometheus.NewDesc("zfs_dataset_used_by_children", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUsedByDataset, prometheus.NewDesc("zfs_dataset_used_by_dataset", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUsedByRefReservation, prometheus.NewDesc("zfs_dataset_used_by_ref_reservation", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUsedBySnapshots, prometheus.NewDesc("zfs_dataset_used_by_snapshots", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetReferenced, prometheus.NewDesc("zfs_dataset_referenced", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetRefCompressRatio, prometheus.NewDesc("zfs_dataset_ref_compress_ratio", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetLogicalReferenced, prometheus.NewDesc("zfs_dataset_logical_referenced", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetLogicalUsed, prometheus.NewDesc("zfs_dataset_logical_used", "", []string{"name", "pool"}, nil))

	describe(ch, &c.datasetWrites, prometheus.NewDesc("zfs_dataset_writes", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetNWritten, prometheus.NewDesc("zfs_dataset_nwritten", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetReads, prometheus.NewDesc("zfs_dataset_reads", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetNRead, prometheus.NewDesc("zfs_dataset_nread", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetUnlinks, prometheus.NewDesc("zfs_dataset_nunlinks", "", []string{"name", "pool"}, nil))
	describe(ch, &c.datasetNUnlinked, prometheus.NewDesc("zfs_dataset_nunlinked", "", []string{"name", "pool"}, nil))
}

func (c *zfsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.describe(&ch)
}

func (c *zfsCollector) handleVdev(ch *chan<- prometheus.Metric, pool string, vdevNamePrefix string, vdev *vdev) error {
	vdevName := ""
	if vdev.path != "" {
		p := path.Base(vdev.path)
		vdevName = p
	} else {
		if vdev.vdevType == "root" {
			vdevName = pool
		} else {
			vdevName = fmt.Sprintf("%s-%d", vdev.vdevType, vdev.id)
		}
	}
	vdevName = vdevNamePrefix + vdevName

	labels := []string{pool, vdevName, vdev.vdevType}

	for _, vdevState := range ioctl.VDevStates {
		val := 0.0
		if vdevState == vdev.vdevStats.state {
			val = 1.0
		}
		metric, err := prometheus.NewConstMetric(c.poolVdevState, prometheus.GaugeValue, val, pool, vdevName, vdev.vdevType, vdevState)
		if err != nil {
			return err
		}
		if ch != nil {
			*ch <- metric
		}
	}

	if err := export(ch, c.poolVdevAllocSpaceDesc, prometheus.GaugeValue, float64(vdev.vdevStats.alloc), labels); err != nil {
		return err
	}
	if err := export(ch, c.poolVdevTotalSpaceDesc, prometheus.GaugeValue, float64(vdev.vdevStats.size), labels); err != nil {
		return err
	}
	// if err := export(ch, c.poolVdevDefSpaceDesc, prometheus.GaugeValue, vdev.DefSpace, labels); err != nil {
	// 	return err
	// }
	// if err := export(ch, c.poolVdevRepDevSizeDesc, prometheus.GaugeValue, vdev.RepDevSize, labels); err != nil {
	// 	return err
	// }
	// if err := export(ch, c.poolVdevPhysSpace, prometheus.GaugeValue, vdev.PhysSpace, labels); err != nil {
	// 	return err
	// }

	if err := export(ch, c.poolVdevReadOps, prometheus.CounterValue, float64(vdev.vdevStats.readOperations), labels); err != nil {
		return err
	}
	if err := export(ch, c.poolVdevReadBytes, prometheus.CounterValue, float64(vdev.vdevStats.bytesRead), labels); err != nil {
		return err
	}
	if err := export(ch, c.poolVdevReadErrors, prometheus.CounterValue, float64(vdev.vdevStats.readErrors), labels); err != nil {
		return err
	}

	if err := export(ch, c.poolVdevWriteOps, prometheus.CounterValue, float64(vdev.vdevStats.writeOperations), labels); err != nil {
		return err
	}
	if err := export(ch, c.poolVdevWriteBytes, prometheus.CounterValue, float64(vdev.vdevStats.bytesWritten), labels); err != nil {
		return err
	}
	if err := export(ch, c.poolVdevWriteErrors, prometheus.CounterValue, float64(vdev.vdevStats.writeErrors), labels); err != nil {
		return err
	}

	if err := export(ch, c.poolVdevChechsumErrors, prometheus.CounterValue, float64(vdev.vdevStats.checksumErrors), labels); err != nil {
		return err
	}
	// if err := export(ch, c.poolVdevSlowIos, prometheus.CounterValue, vdev.SlowIos, labels); err != nil {
	// 	return err
	// }

	for _, child := range vdev.children {
		err := c.handleVdev(ch, pool, vdevName+"/", child)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *zfsCollector) handleDataset(ch *chan<- prometheus.Metric, pool string, name string, props *datasetProps) error {
	labels := []string{name, pool}

	if err := export(ch, c.datasetAvailable, prometheus.GaugeValue, float64(props.available), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetCompressRatio, prometheus.GaugeValue, float64(props.compressratio)/100, labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetUsed, prometheus.GaugeValue, float64(props.used), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetUsedByChildren, prometheus.GaugeValue, float64(props.usedbychildren), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetUsedByDataset, prometheus.GaugeValue, float64(props.usedbydataset), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetUsedByRefReservation, prometheus.GaugeValue, float64(props.usedbyrefreservation), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetUsedBySnapshots, prometheus.GaugeValue, float64(props.usedbysnapshots), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetReferenced, prometheus.GaugeValue, float64(props.referenced), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetRefCompressRatio, prometheus.GaugeValue, float64(props.refcompressratio)/100, labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetLogicalReferenced, prometheus.GaugeValue, float64(props.logicalreferenced), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetLogicalUsed, prometheus.GaugeValue, float64(props.logicalused), labels); err != nil {
		return err
	}

	if err := export(ch, c.datasetWrites, prometheus.CounterValue, float64(props.writes), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetNWritten, prometheus.CounterValue, float64(props.nwritten), labels); err != nil {
		return err
	}

	if err := export(ch, c.datasetReads, prometheus.CounterValue, float64(props.reads), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetNRead, prometheus.CounterValue, float64(props.nread), labels); err != nil {
		return err
	}

	if err := export(ch, c.datasetUnlinks, prometheus.CounterValue, float64(props.nunlinks), labels); err != nil {
		return err
	}
	if err := export(ch, c.datasetNUnlinked, prometheus.CounterValue, float64(props.nunlinked), labels); err != nil {
		return err
	}

	return nil
}

type datasetProps struct {
	objsetid uint64

	available            uint64
	compressratio        uint64
	used                 uint64
	usedbychildren       uint64
	usedbydataset        uint64
	usedbyrefreservation uint64
	usedbysnapshots      uint64
	referenced           uint64
	refcompressratio     uint64
	logicalreferenced    uint64
	logicalused          uint64

	hasKStats bool
	writes    uint64
	nwritten  uint64
	reads     uint64
	nread     uint64
	nunlinks  uint64
	nunlinked uint64
}

func (d *datasetProps) parseValue(r *nvlist.NVListReader, propName string) error {
	for {
		token, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if r.Name() == "value" {
			switch propName {
			case "objsetid":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for objsetid")
				}
				d.objsetid = r.UInt64()
			case "available":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for available")
				}
				d.available = r.UInt64()
			case "compressratio":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for compressratio")
				}
				d.compressratio = r.UInt64()
			case "used":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for used")
				}
				d.used = r.UInt64()
			case "usedbychildren":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for usedbychildren")
				}
				d.usedbychildren = r.UInt64()
			case "usedbydataset":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for usedbydataset")
				}
				d.usedbydataset = r.UInt64()
			case "usedbyrefreservation":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for usedbyrefreservation")
				}
				d.usedbyrefreservation = r.UInt64()
			case "usedbysnapshots":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for usedbysnapshots")
				}
				d.usedbysnapshots = r.UInt64()
			case "referenced":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for referenced")
				}
				d.referenced = r.UInt64()
			case "refcompressratio":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for refcompressratio")
				}
				d.refcompressratio = r.UInt64()
			case "logicalreferenced":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for logicalreferenced")
				}
				d.logicalreferenced = r.UInt64()
			case "logicalused":
				if token != nvlist.TypeUint64 {
					return fmt.Errorf("invalid type for logicalused")
				}
				d.logicalused = r.UInt64()
			}
		}
	}
	return nil
}

func (d *datasetProps) parseProps(r *nvlist.NVListReader) error {
	for {
		token, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if token == nvlist.TypeNvlist {
			err = d.parseValue(r, r.Name())
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("expected nvlist, got %v", token)
		}
	}

	return nil
}

func (d *datasetProps) parseKStat(r *kstat.KStatReader) error {
	for {
		name, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch name {
		case "writes":
			d.writes, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"writes\" row: %w", err)
			}
		case "nwritten":
			d.nwritten, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"nwritten\" row: %w", err)
			}
		case "reads":
			d.reads, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"reads\" row: %w", err)
			}
		case "nread":
			d.nread, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"nread\" row: %w", err)
			}
		case "nunlinks":
			d.nunlinks, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"nunlinks\" row: %w", err)
			}
		case "nunlinked":
			d.nunlinked, err = r.RowDataAsUInt64()
			if err != nil {
				return fmt.Errorf("error reading \"nunlinked\" row: %w", err)
			}
		}
	}

	return nil
}

type vdevStats struct {
	alloc           uint64
	free            uint64
	size            uint64
	readOperations  uint64
	bytesRead       uint64
	readErrors      uint64
	writeOperations uint64
	bytesWritten    uint64
	writeErrors     uint64
	checksumErrors  uint64
	fragmentation   uint64
	state           string
}

func parseVdevStats(vdevStats []uint64) (s vdevStats) {
	s.alloc = vdevStats[ioctl.VDevStats_vs_alloc]
	s.free = vdevStats[ioctl.VDevStats_vs_space] - vdevStats[ioctl.VDevStats_vs_alloc]
	s.size = vdevStats[ioctl.VDevStats_vs_space]

	s.readOperations = vdevStats[ioctl.VDevStats_vs_ops_read]
	s.bytesRead = vdevStats[ioctl.VDevStats_vs_bytes_read]
	s.readErrors = vdevStats[ioctl.VDevStats_vs_read_errors]

	s.writeOperations = vdevStats[ioctl.VDevStats_vs_ops_write]
	s.bytesWritten = vdevStats[ioctl.VDevStats_vs_bytes_write]
	s.writeErrors = vdevStats[ioctl.VDevStats_vs_write_errors]

	s.checksumErrors = vdevStats[ioctl.VDevStats_vs_checksum_errors]
	s.fragmentation = vdevStats[ioctl.VDevStats_vs_fragmentation]

	s.state = ioctl.VDevStateString(vdevStats[ioctl.VDevStats_vs_state], vdevStats[ioctl.VDevStats_vs_aux])

	return
}

func parseVdevs(vdevReader *nvlist.NVListReader) (*vdev, error) {
	vdev := &vdev{}

	for {
		token, err := vdevReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Parse the vdev class to be able to add it as a label upon export:
		// https://sourcegraph.com/github.com/openzfs/zfs@3862ebbf1fe1f8755f9956a8eaecaefc428c8f31/-/blob/cmd/zpool/zpool_main.c?L1208-1247

		if vdevReader.Name() == "children" {
			if token != nvlist.TypeNvlistArray {
				return nil, fmt.Errorf("invalid children")
			}
			numChildren := vdevReader.NumElements()
			for i := 0; i < numChildren; i++ {
				child, err := parseVdevs(vdevReader)
				if err != nil {
					return nil, err
				}
				vdev.children = append(vdev.children, child)
			}
		} else if vdevReader.Name() == "type" {
			if token != nvlist.TypeString {
				return nil, fmt.Errorf("invalid type")
			}
			s, err := vdevReader.String()
			if err != nil {
				return nil, err
			}

			vdev.vdevType = strings.Clone(s)
		} else if vdevReader.Name() == "id" {
			if token != nvlist.TypeUint64 {
				return nil, fmt.Errorf("invalid type")
			}
			vdev.id = vdevReader.UInt64()
		} else if vdevReader.Name() == "path" {
			if token != nvlist.TypeString {
				return nil, fmt.Errorf("invalid type")
			}
			s, err := vdevReader.String()
			if err != nil {
				return nil, err
			}
			vdev.path = strings.Clone(s)
		} else if vdevReader.Name() == "vdev_stats" {
			if token != nvlist.TypeUint64Array {
				return nil, fmt.Errorf("invalid vdev_stats")
			}
			vdev.vdevStats = parseVdevStats(vdevReader.UInt64Array())
		} else if token == nvlist.TypeNvlist || token == nvlist.TypeNvlistArray {
			err = vdevReader.Skip()
			if err != nil {
				return nil, err
			}
		}
	}

	return vdev, nil
}

type vdev struct {
	vdevType string
	id       uint64
	path     string
	children []*vdev

	vdevStats vdevStats
}

func (c *zfsCollector) handlePool(ch *chan<- prometheus.Metric, poolName string) error {
	cmd := ioctl.Cmd{}
	cmd.SetName(poolName)
	resp := make([]byte, 256*1024)
	err := c.zfsHandle.Ioctl(ioctl.ZFS_IOC_POOL_STATS, &cmd, nil, nil, &resp)
	if err != nil {
		return err
	}

	var state string
	var errorCount uint64
	var vdev *vdev

	poolStatsReader := nvlist.NVListReader{Data: resp}

	for {
		token, err := poolStatsReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if poolStatsReader.Name() == "state" {
			if token != nvlist.TypeUint64 {
				return fmt.Errorf("invalid state")
			}
			state = ioctl.PoolStateString(poolStatsReader.UInt64())
		} else if poolStatsReader.Name() == "error_count" {
			if token != nvlist.TypeUint64 {
				return fmt.Errorf("invalid error_count")
			}
			errorCount = poolStatsReader.UInt64()
		} else if poolStatsReader.Name() == "error_count" {
			if token != nvlist.TypeUint64 {
				return fmt.Errorf("invalid error_count")
			}
			errorCount = poolStatsReader.UInt64()
		} else if poolStatsReader.Name() == "vdev_tree" {
			if token != nvlist.TypeNvlist {
				return fmt.Errorf("invalid vdev_tree")
			}
			vdev, err = parseVdevs(&poolStatsReader)
			if err != nil {
				return err
			}
		} else if token == nvlist.TypeNvlist || token == nvlist.TypeNvlistArray {
			err = poolStatsReader.Skip()
			if err != nil {
				return err
			}
		}
	}

	for _, poolState := range ioctl.PoolStates {
		val := 0.0
		if poolState == state {
			val = 1.0
		}
		metric, err := prometheus.NewConstMetric(c.poolState, prometheus.GaugeValue, val, poolName, poolState)
		if err != nil {
			return err
		}
		if ch != nil {
			*ch <- metric
		}
	}

	metric, err := prometheus.NewConstMetric(c.poolErrorCount, prometheus.CounterValue, float64(errorCount), poolName)
	if err != nil {
		return err
	}
	if ch != nil {
		*ch <- metric
	}

	err = c.handleVdev(ch, poolName, "", vdev)
	if err != nil {
		return err
	}

	var findDatasetsRecursive func(prefix string) error
	findDatasetsRecursive = func(prefix string) error {
		cookie := uint64(0)
		for {
			cmd.Clear()
			cmd.SetName(prefix)
			cmd.Cookie = cookie
			err = c.zfsHandle.Ioctl(ioctl.ZFS_IOC_DATASET_LIST_NEXT, &cmd, nil, nil, &resp)
			if err == unix.ESRCH {
				return nil
			}
			if err != nil {
				return fmt.Errorf("error calling dataset list next: %w", err)
			}
			name := cmd.GetName()
			cookie = cmd.Cookie

			datasetPropsReader := nvlist.NVListReader{Data: resp}
			props := datasetProps{}
			err = props.parseProps(&datasetPropsReader)
			if err != nil {
				return err
			}

			kstatData, err := os.ReadFile(fmt.Sprintf("/proc/spl/kstat/zfs/%s/objset-0x%x", poolName, props.objsetid))
			if err != nil {
				// Either kstats not supported or dataset not mounted...
				if !os.IsNotExist(err) {
					return fmt.Errorf("error reading kstats for %q (objset %v): %w", name, props.objsetid, err)
				}
			} else {
				r := kstat.KStatReader{
					Data: kstatData,
				}
				err := props.parseKStat(&r)
				if err != nil {
					return fmt.Errorf("error parsing kstats for %q (objset %v): %w", name, props.objsetid, err)
				}
			}

			err = c.handleDataset(ch, poolName, name, &props)
			if err != nil {
				return err
			}

			err = findDatasetsRecursive(name)
			if err != nil {
				return err
			}
		}
	}
	err = findDatasetsRecursive(poolName)
	if err != nil {
		return err
	}

	return nil
}

func (c *zfsCollector) collect(ch *chan<- prometheus.Metric) error {
	cmd := ioctl.Cmd{}
	resp := make([]byte, 256*1024)
	err := c.zfsHandle.Ioctl(ioctl.ZFS_IOC_POOL_CONFIGS, &cmd, nil, nil, &resp)
	if err != nil {
		return err
	}

	poolConfigsReader := nvlist.NVListReader{Data: resp}
	for {
		token, err := poolConfigsReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if token != nvlist.TypeNvlist {
			return fmt.Errorf("invalid pool configs")
		}

		poolName := strings.Clone(poolConfigsReader.Name())
		err = c.handlePool(ch, poolName)
		if err != nil {
			return err
		}

		err = poolConfigsReader.Skip()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *zfsCollector) Collect(ch chan<- prometheus.Metric) {
	err := c.collect(&ch)
	if err != nil {
		slog.Error("error collecting and exporting zfs metrics", "error", err)
	}
}

func setup(reg *prometheus.Registry) error {
	zfsHandle, err := ioctl.NewZFSHandle()
	if err != nil {
		return fmt.Errorf("error creating zfs handle: %w", err)
	}

	err = reg.Register(&zfsCollector{zfsHandle: zfsHandle})
	if err != nil {
		return fmt.Errorf("error registering zfs collector: %w", err)
	}

	err = reg.Register(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	if err != nil {
		return fmt.Errorf("error registering process collector: %w", err)
	}
	err = reg.Register(
		collectors.NewGoCollector(),
	)
	if err != nil {
		return fmt.Errorf("error registering go collector: %w", err)
	}
	return nil
}

func main() {
	flag.Parse()

	reg := prometheus.NewPedanticRegistry()
	err := setup(reg)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
