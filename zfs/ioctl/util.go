package ioctl

const (
	VDevStats_vs_timestamp              = iota /* time since vdev load	*/
	VDevStats_vs_state                         /* vdev state		*/
	VDevStats_vs_aux                           /* see vdev_aux_t	*/
	VDevStats_vs_alloc                         /* space allocated	*/
	VDevStats_vs_space                         /* total capacity	*/
	VDevStats_vs_dspace                        /* deflated capacity	*/
	VDevStats_vs_rsize                         /* replaceable dev size */
	VDevStats_vs_esize                         /* expandable dev size */
	VDevStats_vs_ops_null                      /* ignore */
	VDevStats_vs_ops_read                      /* operation count: read	*/
	VDevStats_vs_ops_write                     /* operation count: write	*/
	VDevStats_vs_ops_free                      /* operation count: free	*/
	VDevStats_vs_ops_claim                     /* operation count: claim	*/
	VDevStats_vs_ops_flush                     /* operation count: flush	*/
	VDevStats_vs_ops_trim                      /* operation count: trim	*/
	VDevStats_vs_bytes_read                    /* bytes: read	*/
	VDevStats_vs_bytes_write                   /* bytes: write	*/
	VDevStats_vs_bytes_free                    /* bytes: free	*/
	VDevStats_vs_bytes_claim                   /* bytes: claim	*/
	VDevStats_vs_bytes_flush                   /* bytes: flush	*/
	VDevStats_vs_bytes_trim                    /* bytes: trim	*/
	VDevStats_vs_read_errors                   /* read errors		*/
	VDevStats_vs_write_errors                  /* write errors		*/
	VDevStats_vs_checksum_errors               /* checksum errors	*/
	VDevStats_vs_initialize_errors             /* initializing errors	*/
	VDevStats_vs_self_healed                   /* self-healed bytes	*/
	VDevStats_vs_scan_removing                 /* removing?	*/
	VDevStats_vs_scan_processed                /* scan processed bytes	*/
	VDevStats_vs_fragmentation                 /* device fragmentation */
	VDevStats_vs_initialize_bytes_done         /* bytes initialized */
	VDevStats_vs_initialize_bytes_est          /* total bytes to initialize */
	VDevStats_vs_initialize_state              /* vdev_initializing_state_t */
	VDevStats_vs_initialize_action_time        /* time_t */
	VDevStats_vs_checkpoint_space              /* checkpoint-consumed space */
	VDevStats_vs_resilver_deferred             /* resilver deferred	*/
	VDevStats_vs_slow_ios                      /* slow IOs */
	VDevStats_vs_trim_errors                   /* trimming errors	*/
	VDevStats_vs_trim_notsup                   /* supported by device */
	VDevStats_vs_trim_bytes_done               /* bytes trimmed */
	VDevStats_vs_trim_bytes_est                /* total bytes to trim */
	VDevStats_vs_trim_state                    /* vdev_trim_state_t */
	VDevStats_vs_trim_action_time              /* time_t */
	VDevStats_vs_rebuild_processed             /* bytes rebuilt */
	VDevStats_vs_configured_ashift             /* TLV vdev_ashift */
	VDevStats_vs_logical_ashift                /* vdev_logical_ashift  */
	VDevStats_vs_physical_ashift               /* vdev_physical_ashift */
	VDevStats_vs_noalloc                       /* allocations halted?	*/
	VDevStats_vs_pspace                        /* physical capacity */
	VDevStats_vs_dio_verify_errors             /* DIO write verify errors */
)

const (
	vdevState_UNKNOWN   = iota /* Uninitialized vdev			*/
	vdevState_CLOSED           /* Not currently open			*/
	vdevState_OFFLINE          /* Not allowed to open			*/
	vdevState_REMOVED          /* Explicitly removed from system	*/
	vdevState_CANT_OPEN        /* Tried to open, but failed		*/
	vdevState_FAULTED          /* External request to fault device	*/
	vdevState_DEGRADED         /* Replicated vdev with unhealthy kids	*/
	vdevState_HEALTHY          /* Presumed good			*/
)

const (
	vdevAux_NONE             = iota /* no error				*/
	vdevAux_OPEN_FAILED             /* ldi_open_*() or vn_open() failed	*/
	vdevAux_CORRUPT_DATA            /* bad label or disk contents		*/
	vdevAux_NO_REPLICAS             /* insufficient number of replicas	*/
	vdevAux_BAD_GUID_SUM            /* vdev guid sum doesn't match		*/
	vdevAux_TOO_SMALL               /* vdev size is too small		*/
	vdevAux_BAD_LABEL               /* the label is OK but invalid		*/
	vdevAux_VERSION_NEWER           /* on-disk version is too new		*/
	vdevAux_VERSION_OLDER           /* on-disk version is too old		*/
	vdevAux_UNSUP_FEAT              /* unsupported features			*/
	vdevAux_SPARED                  /* hot spare used in another pool	*/
	vdevAux_ERR_EXCEEDED            /* too many errors			*/
	vdevAux_IO_FAILURE              /* experienced I/O failure		*/
	vdevAux_BAD_LOG                 /* cannot read log chain(s)		*/
	vdevAux_EXTERNAL                /* external diagnosis or forced fault	*/
	vdevAux_SPLIT_POOL              /* vdev was split off into another pool	*/
	vdevAux_BAD_ASHIFT              /* vdev ashift is invalid		*/
	vdevAux_EXTERNAL_PERSIST        /* persistent forced fault	*/
	vdevAux_ACTIVE                  /* vdev active on a different host	*/
	vdevAux_CHILDREN_OFFLINE        /* all children are offline		*/
	vdevAux_ASHIFT_TOO_BIG          /* vdev's min block size is too large   */
)

const (
	poolState_ACTIVE             = iota /* In active use		*/
	poolState_EXPORTED                  /* Explicitly exported		*/
	poolState_DESTROYED                 /* Explicitly destroyed		*/
	poolState_SPARE                     /* Reserved for hot spare use	*/
	poolState_L2CACHE                   /* Level 2 ARC device		*/
	poolState_UNINITIALIZED             /* Internal spa_t state		*/
	poolState_UNAVAIL                   /* Internal libzfs state	*/
	poolState_POTENTIALLY_ACTIVE        /* Internal libzfs state	*/
)

var (
	PoolStates = [...]string{
		"ACTIVE",
		"EXPORTED",
		"DESTROYED",
		"SPARE",
		"L2CACHE",
		"UNINITIALIZED",
		"UNAVAIL",
		"POTENTIALLY_ACTIVE",
		"UNKNOWN",
	}
	VDevStates = [...]string{
		"OFFLINE",
		"REMOVED",
		"FAULTED",
		"SPLIT",
		"UNAVAIL",
		"DEGRADED",
		"ONLINE",
		"UNKNOWN",
	}
)

const vdevState_ONLINE = vdevState_HEALTHY

func PoolStateString(state uint64) string {
	switch state {
	case poolState_ACTIVE:
		return "ACTIVE"
	case poolState_EXPORTED:
		return "EXPORTED"
	case poolState_DESTROYED:
		return "DESTROYED"
	case poolState_SPARE:
		return "SPARE"
	case poolState_L2CACHE:
		return "L2CACHE"
	case poolState_UNINITIALIZED:
		return "UNINITIALIZED"
	case poolState_UNAVAIL:
		return "UNAVAIL"
	case poolState_POTENTIALLY_ACTIVE:
		return "POTENTIALLY_ACTIVE"
	}

	return "UNKNOWN"
}

func VDevStateString(state uint64, aux uint64) string {
	switch state {
	case vdevState_CLOSED:
	case vdevState_OFFLINE:
		return "OFFLINE"
	case vdevState_REMOVED:
		return "REMOVED"
	case vdevState_CANT_OPEN:
		if aux == vdevAux_CORRUPT_DATA || aux == vdevAux_BAD_LOG {
			return "FAULTED"
		} else if aux == vdevAux_SPLIT_POOL {
			return "SPLIT"
		} else {
			return "UNAVAIL"
		}
	case vdevState_FAULTED:
		return "FAULTED"
	case vdevState_DEGRADED:
		return "DEGRADED"
	case vdevState_HEALTHY:
		return "ONLINE"
	}

	return "UNKNOWN"
}
