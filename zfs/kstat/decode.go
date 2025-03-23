package kstat

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"unsafe"
)

type KStatHeader struct {
	Type uint8
}

type KStatReader struct {
	Data []byte
	pos  int

	Header KStatHeader

	rowName string
	rowType string
	rowData string
}

func (r *KStatReader) readUntilExclude(b byte) (string, error) {
	data := r.Data
	pos := r.pos

	for {
		if pos >= len(data) {
			return "", io.EOF
		}
		if data[pos] == b {
			break
		}
		pos++
	}
	d := data[r.pos:pos]
	// Eat the character we want to exclude
	pos++
	r.pos = pos

	s := unsafe.String(unsafe.SliceData(d), len(d))
	return s, nil
}

func (r *KStatReader) readUntilExcludeIgnoringPrefix(b byte, ignorePrefix byte) (string, error) {
	data := r.Data
	pos := r.pos

	for {
		if pos >= len(data) {
			return "", io.EOF
		}
		if data[pos] != ignorePrefix {
			break
		}
		pos++
	}

	r.pos = pos

	return r.readUntilExclude(b)
}

func (r *KStatReader) readHeader() error {
	kidBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading kid from header: %w", err)
	}

	typeBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading type from header: %w", err)
	}
	t, err := strconv.ParseInt(typeBytes, 10, 32)
	if err != nil {
		return fmt.Errorf("error parsing type from header: %w", err)
	}
	r.Header.Type = uint8(t)

	flagsBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading flags from header: %w", err)
	}
	ndataBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading kid from header: %w", err)
	}
	dataSizeBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading kid from header: %w", err)
	}
	crTimeBytes, err := r.readUntilExclude(' ')
	if err != nil {
		return fmt.Errorf("error reading kid from header: %w", err)
	}
	snapTimeBytes, err := r.readUntilExclude('\n')
	if err != nil {
		return fmt.Errorf("error reading kid from header: %w", err)
	}

	runtime.KeepAlive(kidBytes)
	runtime.KeepAlive(flagsBytes)
	runtime.KeepAlive(ndataBytes)
	runtime.KeepAlive(dataSizeBytes)
	runtime.KeepAlive(crTimeBytes)
	runtime.KeepAlive(snapTimeBytes)

	return nil
}

func (r *KStatReader) readColumnHeaders() error {
	c1, err := r.readUntilExcludeIgnoringPrefix(' ', ' ')
	if err != nil {
		return fmt.Errorf("error reading column header")
	}
	if c1 != "name" {
		return fmt.Errorf("unexpected column header: want \"name\", got %q", c1)
	}
	c2, err := r.readUntilExcludeIgnoringPrefix(' ', ' ')
	if err != nil {
		return fmt.Errorf("error reading column header")
	}
	if c2 != "type" {
		return fmt.Errorf("unexpected column header: want \"type\", got %q", c1)
	}
	c3, err := r.readUntilExcludeIgnoringPrefix('\n', ' ')
	if err != nil {
		return fmt.Errorf("error reading column header")
	}
	if c3 != "data" {
		return fmt.Errorf("unexpected column header: want \"data\", got %q", c1)
	}

	return nil
}

func (r *KStatReader) Next() (string, error) {
	if r.pos == 0 {
		err := r.readHeader()
		if err != nil {
			return "", err
		}

		if r.Header.Type != 1 {
			return "", fmt.Errorf("KStatReader only supports kstat that are of type key-value pair")
		}

		err = r.readColumnHeaders()
		if err != nil {
			return "", fmt.Errorf("error reading column headers: %w", err)
		}
	}

	if r.pos == len(r.Data) {
		return "", io.EOF
	}

	var err error
	r.rowName, err = r.readUntilExcludeIgnoringPrefix(' ', ' ')
	if err != nil {
		return "", fmt.Errorf("error reading value of column name")
	}
	r.rowType, err = r.readUntilExcludeIgnoringPrefix(' ', ' ')
	if err != nil {
		return "", fmt.Errorf("error reading value of column type")
	}
	r.rowData, err = r.readUntilExcludeIgnoringPrefix('\n', ' ')
	if err != nil {
		return "", fmt.Errorf("error reading value of column data")
	}

	return r.rowName, nil
}

func (r *KStatReader) RowData() string {
	return r.rowData
}

func (r *KStatReader) RowDataAsUInt64() (uint64, error) {
	i, err := strconv.ParseUint(r.RowData(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing row data as uint64: %w", err)
	}
	return i, nil
}
